package conc

import "log"

type ReaderFunc[R any] func() (R, error)

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
	log.Println("Cleaning up reader...")
	defer log.Println("Finished cleaning up reader...")
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
				newMessage, err := rc.Read()
				if rc.msgChannel != nil {
					rc.msgChannel <- Message[R]{
						Value: newMessage,
						Error: err,
					}
				}
				if err != nil {
					log.Println("Read Error: ", err)
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
