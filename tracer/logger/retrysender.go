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
	return s.retry("Open", s.Sender.Open)
}
func (s *RetrySender) Close() error {
	return s.retry("Close", s.Sender.Close)
}

func (s *RetrySender) SendSymbols(data *logutil.SymbolsData) error {
	return s.retrySend("SendSymbols", func() error {
		// try to send
		return s.Sender.SendSymbols(data)
	})
}

// SendLog sends a RawFuncLog.
// if occur the any error, retry to send after re-open.
func (s *RetrySender) SendLog(raw *logutil.RawFuncLog) error {
	return s.retrySend("SendLog", func() error {
		return s.Sender.SendLog(raw)
	})
}

// retry is automatically retry until reached to retry limit or fn() is succeed.
func (s *RetrySender) retry(funcName string, fn func() error) error {
	var err error
	for i := 0; i < s.MaxRetry; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		log.Printf("failed to Sender.%s() on RetrySender.%s(): %s", funcName, funcName, err)
		s.sleep(i)
	}
	return err
}

// retrySend is automatically retry until reached to retry limit or send() is succeed.
// When send() is not succeed, it will reopen the sender.
func (s *RetrySender) retrySend(funcName string, send func() error) error {
	var senderr error
	for i := 0; i < s.MaxRetry; i++ {
		senderr = send()
		if senderr == nil {
			return nil
		}
		log.Printf("failed to RetrySender.%s(): %s", funcName, senderr)

		// try to close.
		// if occurs any error, we print of logging message.
		closeerr := s.Close()
		if closeerr != nil {
			log.Printf("failed to Sender.Close() on RetrySender.%s(): %s", funcName, closeerr)
		}

		// try to re-open.
		openerr := s.Open()
		if openerr != nil {
			log.Panicf("failed to Sender.Open() on RetrySender.%s(): %s", funcName, openerr)
			return openerr
		}

		s.sleep(i)
	}
	return senderr
}

func (s *RetrySender) sleep(retryCount int) time.Duration {
	if s.RetryInterval == 0 {
		return defaultRetryInterval
	}
	return s.RetryInterval
}
