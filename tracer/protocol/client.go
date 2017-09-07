package protocol

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strings"

	"time"
)

type ClientHandler struct {
	Connected    func()
	Disconnected func()

	Error func(error)

	StartTrace func(*StartTraceCmdArgs)
	StopTrace  func(*StopTraceCmdArgs)
}

type Client struct {
	// "unix:///path/to/socket/file" or "tcp://host:port"
	Addr    string
	Handler ClientHandler

	AppName         string
	Version         string
	Secret          string
	MaxBufferedMsgs int
	Timeout         time.Duration
	PingInterval    time.Duration

	conn      net.Conn
	cancel    context.CancelFunc
	workerCtx context.Context

	writeChan chan interface{}
}

func New() {
	// TODO
}

func (c *Client) Connect() error {
	var proto string
	var url string

	switch {
	case strings.HasPrefix(c.Addr, "unix://"):
		url = strings.TrimPrefix(c.Addr, "unix://")
		proto = "unix"
	case strings.HasPrefix(c.Addr, "tcp://"):
		url = strings.TrimPrefix(c.Addr, "tcp://")
		proto = "tcp"
	default:
		return errors.New("Invalid protocol")
	}

	conn, err := net.Dial(proto, url)
	if err != nil {
		return err
	}
	c.conn = conn
	c.workerCtx, c.cancel = context.WithCancel(context.Background())
	if c.MaxBufferedMsgs <= 0 {
		c.MaxBufferedMsgs = DefaultMaxBufferedMsgs
	}
	c.writeChan = make(chan interface{}, c.MaxBufferedMsgs)
	if c.Timeout == time.Duration(0) {
		c.Timeout = DefaultTimeout
	}
	if c.PingInterval == time.Duration(0) {
		c.PingInterval = DefaultPingInterval
	}

	go c.worker()
	c.Handler.Connected()
	return nil
}

func (c *Client) Send(msgType MessageType, data interface{}) {
	c.writeChan <- &MessageHeader{
		MessageType: msgType,
		Messages:    1,
	}
	c.writeChan <- data
}

func (c *Client) Close() error {
	if c.workerCtx != nil {
		c.workerCtx = nil
		c.cancel()
		c.cancel = nil

		close(c.writeChan)

		err := c.conn.Close()
		c.conn = nil
		c.Handler.Disconnected()
		return err
	}
	return nil
}

func (c *Client) worker() {
	errCh := make(chan error)
	shouldStop := false

	isError := func(err error) bool {
		if shouldStop {
			return true
		}
		errCh <- err
		return true
	}

	go func() {
		setReadDeadline := func() {
			c.conn.SetReadDeadline(time.Now().Add(c.Timeout))
		}
		setWriteDeadline := func() {
			c.conn.SetWriteDeadline(time.Now().Add(c.Timeout))
		}

		// initialize
		enc := gob.NewEncoder(c.conn)
		dec := gob.NewDecoder(c.conn)

		setWriteDeadline()
		if isError(enc.Encode(&ClientHeader{
			AppName:       c.AppName,
			ClientSecret:  c.Secret,
			ClientVersion: c.Version,
		})) {
			return
		}

		setReadDeadline()
		serverHeader := ServerHeader{}
		if isError(dec.Decode(&serverHeader)) {
			return
		}
		// TODO: check response

		// initialize process is done
		// start read/write workers

		// start read worker
		go func() {
			for !shouldStop {
				setReadDeadline()
				cmdHeader := &CommandHeader{}
				if isError(dec.Decode(cmdHeader)) {
					return
				}
				if shouldStop {
					return
				}

				var data interface{}
				switch cmdHeader.CommandType {
				case StartTraceCmd:
					data = &StartTraceCmdArgs{}
				case StopTraceCmd:
					data = &StopTraceCmdArgs{}
				default:
					errCh <- errors.New(fmt.Sprintf("Invalid CommandType: %d", cmdHeader.CommandType))
					return
				}

				setReadDeadline()
				if isError(dec.Decode(data)) {
					return
				}
				if shouldStop {
					return
				}

				switch cmdHeader.CommandType {
				case StartTraceCmd:
					c.Handler.StartTrace(data.(*StartTraceCmdArgs))
				case StopTraceCmd:
					c.Handler.StopTrace(data.(*StopTraceCmdArgs))
				default:
					panic("bug")
				}
			}
		}()

		// start ping worker
		go func() {
			for !shouldStop {
				c.Send(PingMsg, &struct{}{})
				time.Sleep(c.PingInterval)
			}
		}()

		// start write worker
		go func() {
			for data := range c.writeChan {
				if shouldStop {
					return
				}

				setWriteDeadline()
				if isError(enc.Encode(data)) {
					return
				}
			}
		}()
	}()

	select {
	case <-c.workerCtx.Done():
		shouldStop = true
		return
	case err := <-errCh:
		c.Handler.Error(err)
		shouldStop = true
		if err := c.Close(); err != nil {
			c.Handler.Error(err)
		}
		return
	}
}