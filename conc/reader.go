package conc

import (
	"errors"
	"log"
	"log/slog"
	"net"
)

// Type of the reader method used by the Reader goroutine primitive.
type ReaderFunc[R any] func() (msg R, err error)

// The typed Reader goroutine which calls a Read method to return data
// over a channel.
type Reader[R any] struct {
	RunnerBase[string]
	msgChannel chan Message[R]
	Read       ReaderFunc[R]
	OnDone     func(r *Reader[R])
}

// Creates a new reader instance.   Just like time.Ticker, this initializer
// also starts the Reader loop.   It is upto the caller to Stop this reader when
// done with.  Not doing so can risk the reader to run indefinitely.
func NewReader[R any](read ReaderFunc[R]) *Reader[R] {
	out := Reader[R]{
		RunnerBase: NewRunnerBase("stop"),
		Read:       read,
		msgChannel: make(chan Message[R]),
	}
	out.start()
	return &out
}

func (r *Reader[R]) DebugInfo() any {
	return map[string]any{
		"base":    r.RunnerBase.DebugInfo(),
		"msgChan": r.msgChannel,
	}
}

// Returns the channel onwhich messages can be received.
func (rc *Reader[R]) RecvChan() <-chan Message[R] {
	return rc.msgChannel
}

func (rc *Reader[R]) start() {
	rc.RunnerBase.start()
	go func() {
		defer rc.cleanup()
		go func() {
			for {
				newMessage, err := rc.Read()
				timedOut := false
				if err != nil {
					nerr, ok := err.(net.Error)
					if ok {
						timedOut = nerr.Timeout()
					}
					log.Println("Net Error, TimedOut, Closed, errors.Is.ErrClosed: ", nerr, timedOut, errors.Is(err, net.ErrClosed), nil)
				}
				if rc.msgChannel != nil && !timedOut && !errors.Is(err, net.ErrClosed) {
					rc.msgChannel <- Message[R]{
						Value: newMessage,
						Error: err,
					}
				}
				if err != nil {
					slog.Debug("Read Error: ", err)
					break
				}
			}
		}()

		// and the actual reader
		for {
			select {
			case <-rc.controlChan:
				// For now only a "kill" can be sent here
				return
			}
		}
	}()
}

func (r *Reader[T]) cleanup() {
	defer log.Println("Cleaned up reader...")
	if r.OnDone != nil {
		r.OnDone(r)
	}
	oldCh := r.msgChannel
	r.msgChannel = nil
	close(oldCh)
	r.RunnerBase.cleanup()
}
