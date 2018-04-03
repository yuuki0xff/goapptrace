package protocol

import (
	"context"
	"log"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yuuki0xff/xtcp"
)

const (
	defaultTimeout = 5 * time.Second
	fastTimeout    = 500 * time.Millisecond
)

func printStack() {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	log.Println(string(buf[:n]))
}

func timeout(t *testing.T, msg string, timeout time.Duration) context.CancelFunc {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	go func() {
		<-ctx.Done()
		switch ctx.Err() {
		case context.Canceled:
			// ok
		case context.DeadlineExceeded:
			printStack()
			log.Fatalf("timeout: %s", msg)
		default:
			printStack()
			log.Fatal("bug")
		}
	}()
	return cancel
}
func withTimeout(t *testing.T, msg string, fn func()) {
	cancel := timeout(t, msg, defaultTimeout)
	defer cancel()
	fn()
}
func withFastTimeout(t *testing.T, msg string, fn func()) {
	cancel := timeout(t, msg, fastTimeout)
	defer cancel()
	fn()
}
func mustNotCall(field string) {
	log.Panicf("\"%s\" handler MUST NOT call", field)
}

type serverConnTestCh struct {
	sendCh chan xtcp.Packet
	stopCh chan xtcp.StopMode
}
type serverConnTest struct {
	T       *testing.T
	Handler *ConnHandler
	// ServerConn.isNegotiated field
	IsNegotiated bool
	// ServerFunc calls sc.OnEvent() to checks the server behavior.
	ServerFunc func(sc ServerConn, xc *xtcp.Conn)
	// ClientFunc receives packets from server through some channels.
	ClientFunc  func(ch serverConnTestCh)
	SendHandler func(conn *xtcp.Conn, packet xtcp.Packet)
	StopHandler func(conn *xtcp.Conn, mode xtcp.StopMode)
}

func (sct *serverConnTest) Run() {
	t := sct.T
	withTimeout(t, t.Name(), func() {
		ch := serverConnTestCh{}
		ch.sendCh = make(chan xtcp.Packet)
		ch.stopCh = make(chan xtcp.StopMode)
		sc := ServerConn{
			ID:      ConnID(0),
			Handler: sct.Handler,
			sendHandler: func(conn *xtcp.Conn, packet xtcp.Packet) {
				withFastTimeout(t, "sendHandler", func() {
					ch.sendCh <- packet
				})
			},
			stopHandler: func(conn *xtcp.Conn, mode xtcp.StopMode) {
				withFastTimeout(t, "stopHandler", func() {
					ch.stopCh <- mode
				})
			},
		}
		if sct.SendHandler != nil {
			sc.sendHandler = sct.SendHandler
		}
		if sct.StopHandler != nil {
			sc.stopHandler = sct.StopHandler
		}
		proto := &Proto{}

		var wa sync.WaitGroup
		wa.Add(2)
		go func() {
			defer wa.Done()
			// this goroutine emulates the client.
			if sct.ServerFunc == nil {
				return
			}
			xc := xtcp.NewConn(xtcp.NewOpts(&sc, proto))
			sc.isNegotiated = sct.IsNegotiated
			sct.ServerFunc(sc, xc)
		}()
		go func() {
			defer wa.Done()
			// this goroutine checks server behaviors.
			if sct.ClientFunc == nil {
				return
			}
			sct.ClientFunc(ch)
		}()
		wa.Wait()
	})
}

func TestServer_Listen(t *testing.T) {
	a := assert.New(t)
	withTimeout(t, t.Name(), func() {
		s := Server{}
		a.NoError(s.Listen())
		a.True(strings.HasPrefix(s.ActualAddr(), "tcp://"))
		a.NoError(s.Close())
	})
}
func TestServer_Wait(t *testing.T) {
	a := assert.New(t)
	withTimeout(t, t.Name(), func() {
		waitCh := make(chan bool)

		s := Server{}
		a.NoError(s.Listen())
		go s.Serve()
		time.Sleep(1 * time.Second)
		go func() {
			s.Wait()
			close(waitCh)
		}()

		select {
		case <-time.NewTimer(1 * time.Second).C:
			// ok
		case <-waitCh:
			t.Fatal("canceled: s.Wait() can not wait for the end of s.Listen()")
		}

		a.NoError(s.Close())
		select {
		case <-time.NewTimer(1 * time.Second).C:
			t.Fatal("timeout: s.Wait() could be waiting something forever")
		case <-waitCh:
			// ok
		}
	})
}
func TestServer_getConnID(t *testing.T) {
	a := assert.New(t)
	s := Server{}
	s.init()

	c1 := &xtcp.Conn{}
	c2 := &xtcp.Conn{}

	a.Equal(ConnID(0), s.getConnID(c1))
	a.Equal(ConnID(1), s.getConnID(c2))
}
func TestServer_getServerConn(t *testing.T) {
	a := assert.New(t)
	s := Server{
		NewHandler: func(id ConnID) *ConnHandler {
			return nil
		},
	}
	s.init()

	a.Equal(s.getServerConn(ConnID(0), nil), s.getServerConn(ConnID(0), nil))
	a.NotEqual(s.getServerConn(ConnID(0), nil), s.getServerConn(ConnID(1), nil))
}
func TestServerConn_OnEvent_handshake(t *testing.T) {
	a := assert.New(t)
	var connected bool

	handler := ConnHandler{
		Connected: func() {
			connected = true
		},
	}.SetDefault(mustNotCall)
	sct := serverConnTest{
		T:       t,
		Handler: &handler,
		ServerFunc: func(sc ServerConn, xc *xtcp.Conn) {
			sc.OnEvent(xtcp.EventAccept, xc, nil)
			sc.OnEvent(xtcp.EventRecv, xc, &ClientHelloPacket{
				ProtocolVersion: ProtocolVersion,
			})
		},
		ClientFunc: func(ch serverConnTestCh) {
			pkt := <-ch.sendCh
			log.Println(pkt)
			a.IsType(&ServerHelloPacket{}, pkt)
		},
	}
	sct.Run()
	a.True(connected)
}
func TestServerConn_OnEvent_receivePingPacket(t *testing.T) {
	// PingPacketに対しては、何も反応してはいけない。
	//a := assert.New(t)
	handler := ConnHandler{}.SetDefault(mustNotCall)
	sct := serverConnTest{
		T:            t,
		Handler:      &handler,
		IsNegotiated: true,
		ServerFunc: func(sc ServerConn, xc *xtcp.Conn) {
			sc.OnEvent(xtcp.EventRecv, xc, &PingPacket{})
		},
		SendHandler: func(conn *xtcp.Conn, packet xtcp.Packet) {
			mustNotCall("SendHandler")
		},
	}
	sct.Run()
}

func TestServerConn_OnEvent_receiveShutdownPacket(t *testing.T) {
	a := assert.New(t)
	var disconnected bool
	var errorOccurred bool
	var stopped bool
	handler := ConnHandler{
		Disconnected: func() {
			disconnected = true
		},
		Error: func(err error) {
			errorOccurred = true
		},
	}.SetDefault(mustNotCall)
	sct := serverConnTest{
		T:            t,
		Handler:      &handler,
		IsNegotiated: true,
		ServerFunc: func(sc ServerConn, xc *xtcp.Conn) {
			sc.OnEvent(xtcp.EventRecv, xc, &ShutdownPacket{})
		},
		ClientFunc: func(ch serverConnTestCh) {
			mode := <-ch.stopCh
			a.Equal(xtcp.StopImmediately, mode)
			stopped = true
		},
		SendHandler: func(conn *xtcp.Conn, packet xtcp.Packet) {
			mustNotCall("SendHandler")
		},
	}
	sct.Run()
	a.True(disconnected)
	a.True(errorOccurred)
	a.True(stopped)
}
