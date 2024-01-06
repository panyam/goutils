package conc

import (
	"log"
	"log/slog"
	"net"
)

type ReaderFunc[R any] func() (msg R, err error, closed bool)

type Reader[R any] struct {
	RunnerBase[string]
	msgChannel chan Message[R]
	Read       ReaderFunc[R]
	OnDone     func(r *Reader[R])
}

func NewReader[R any](read ReaderFunc[R]) *Reader[R] {
	out := Reader[R]{
		RunnerBase: NewRunnerBase("stop"),
		Read:       read,
		msgChannel: make(chan Message[R]),
	}
	out.start()
	return &out
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

/**
 * Returns the conn's reader channel.
 */
func (rc *Reader[R]) RecvChan() <-chan Message[R] {
	return rc.msgChannel
}

func (rc *Reader[R]) start() {
	rc.RunnerBase.start()
	go func() {
		defer rc.cleanup()
		go func() {
			for {
				newMessage, err, closed := rc.Read()
				timedOut := false
				if err != nil {
					if nerr, ok := err.(net.Error); ok {
						timedOut = nerr.Timeout()
						slog.Info("Net Error: ", nerr, timedOut)
					}
					slog.Debug("TimedOut, Closed: ", timedOut, closed)
				}
				if rc.msgChannel != nil && !timedOut && !closed {
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
