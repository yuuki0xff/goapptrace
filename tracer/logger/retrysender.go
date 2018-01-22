package logger

import (
	"log"
	"time"

	"github.com/yuuki0xff/goapptrace/tracer/logutil"
)

// RetrySender is automatically retry to all operations.
type RetrySender struct {
	Sender        Sender
	MaxRetry      int
	RetryInterval time.Duration
}

func (s *RetrySender) Open() error {
	return s.autoretry(s.Sender.Open)
}
func (s *RetrySender) Close() error {
	return s.autoretry(s.Sender.Close)
}

// Send's sends Symbols and RawFuncLog.
// if occur the any error, retry to send after re-open.
func (s *RetrySender) Send(symbols *logutil.Symbols, funclog *logutil.RawFuncLog) error {
	return s.autoretry(func() error {
		// try to send
		err := s.Sender.Send(symbols, funclog)
		if err == nil {
			return nil
		}
		log.Printf("failed to RetrySender.Send(): %s", err)

		// try to close.
		// if occurs any error, we print of logging message.
		err = s.Sender.Close()
		if err != nil {
			log.Printf("failed to Sender.Close() on RetrySender.Send(): %s", err)
		}

		// try to re-open.
		return s.Open()
	})
}

// autoretry is automatically retry until reached to retry limit or fn() is succeed.
func (s *RetrySender) autoretry(fn func() error) error {
	var err error
	for i := 0; i < s.MaxRetry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
	}
	return err
}
