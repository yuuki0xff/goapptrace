package protocol

import (
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"net"
	"strings"

	"log"
	"reflect"
	"sync"
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
	workerWg  sync.WaitGroup

	writeChan chan interface{}
}

func (c *Client) Connect() error {
	log.Println("INFO: clinet: connected")
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

	c.workerWg.Add(1)
	go c.worker()
	c.Handler.Connected()
	return nil
}

func (c *Client) Send(msgType MessageType, data interface{}) {
	log.Printf("DEBUG: client: send message type=%+v, data=%+v\n", msgType, data)
	c.writeChan <- &MessageHeader{
		MessageType: msgType,
		Messages:    1,
	}
	c.writeChan <- data
}

func (c *Client) Close() error {
	log.Println("INFO: client: closing a connection")
	defer log.Println("DEBUG: client: closed a connection")
	if c.cancel != nil {
		// request to worker shutdown
		c.cancel()
		c.cancel = nil

		// disallow send new message to server
		close(c.writeChan)

		// wait for worker ended before close TCP connection
		c.workerWg.Wait()
		err := c.conn.Close()
		c.conn = nil
		c.Handler.Disconnected()
		return err
	}
	return nil
}

func (c *Client) worker() {
	log.Println("INFO: client: start worker")
	defer log.Println("INFO client: ended worker")
	defer c.workerWg.Done()
	errCh := make(chan error)
	shouldStop := false

	isErrorNoStop := func(err error) bool {
		if err != nil {
			if isEOF(err) || isBrokenPipe(err) {
				// ignore errors
				errCh <- nil
				return true
			}
			errCh <- err
			return true
		}
		return false
	}
	isError := func(err error) bool {
		if shouldStop {
			return true
		}
		return isErrorNoStop(err)
	}

	c.workerWg.Add(1)
	go func() {
		log.Println("DEBUG: client: start worker launcher")
		defer log.Println("DEBUG: client: stop worker launcher")
		defer c.workerWg.Done()
		log.Println("INFO: client: initialize")

		setReadDeadline := func() {
			if err := c.conn.SetReadDeadline(time.Now().Add(c.Timeout)); err != nil {
				panic(err)
			}
		}
		setWriteDeadline := func() {
			if err := c.conn.SetWriteDeadline(time.Now().Add(c.Timeout)); err != nil {
				panic(err)
			}
		}

		// initialize
		enc := gob.NewEncoder(c.conn)
		dec := gob.NewDecoder(c.conn)

		log.Println("DEBUG: client: send client header")
		setWriteDeadline()
		if isErrorNoStop(enc.Encode(&ClientHeader{
			AppName:       c.AppName,
			ClientSecret:  c.Secret,
			ClientVersion: c.Version,
		})) {
			return
		}
		log.Println("DEBUG: client: send client header ... done")

		log.Println("DEBUG: client: read server response")
		setReadDeadline()
		serverHeader := ServerHeader{}
		if isErrorNoStop(dec.Decode(&serverHeader)) {
			return
		}
		log.Println("DEBUG: client: read server response ... done")
		// TODO: check response

		// initialize process is done
		// start read/write workers
		log.Println("DEBUG: client: initialize ... done")

		// start read worker
		c.workerWg.Add(1)
		go func() {
			log.Println("DEBUG: client: start read worker")
			defer log.Println("DEBUG: client: stop read worker")
			defer c.workerWg.Done()
			for !shouldStop {
				setReadDeadline()
				cmdHeader := &CommandHeader{}
				if isError(dec.Decode(cmdHeader)) {
					return
				}

				var data interface{}
				switch cmdHeader.CommandType {
				case PingCmd:
					data = &PingCmdArgs{}
				case ShutdownCmd:
					data = &ShutdownCmdArgs{}
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

				switch cmdHeader.CommandType {
				case PingCmd:
					// do nothing
				case ShutdownCmd:
					// do nothing
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
		c.workerWg.Add(1)
		go func() {
			log.Println("DEBUG: client: start ping worker")
			defer log.Println("DEBUG: client: stop ping worker")
			defer c.workerWg.Done()

			for !shouldStop {
				log.Println("DEBUG: client: send ping message")
				c.Send(PingMsg, &PingMsgData{})
				time.Sleep(c.PingInterval)
			}
		}()

		// start write worker
		c.workerWg.Add(1)
		go func() {
			log.Println("DEBUG: client: start write worker")
			defer log.Println("DEBUG: client: stop write worker")
			defer c.workerWg.Done()
			// will be closing c.writeChan by c.Close() when occurred shutdown request.
			// so, this worker should not check 'shouldStop' variable.
			for data := range c.writeChan {
				log.Printf("DEBUG: client: send %s message: %+v\n", reflect.TypeOf(data).String(), data)
				setWriteDeadline()
				if isErrorNoStop(enc.Encode(data)) {
					return
				}
			}
		}()
	}()

	select {
	case <-c.workerCtx.Done():
		// do nothing
	case err := <-errCh:
		if err != nil {
			c.Handler.Error(err)
		}
	}
	// shutdown other workers
	shouldStop = true
}
