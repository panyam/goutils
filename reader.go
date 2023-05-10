package conc

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
	if r.OnDone != nil {
		r.OnDone(r)
	}
	close(r.msgChannel)
	r.msgChannel = nil
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
				rc.msgChannel <- Message[R]{
					Value: newMessage,
					Error: err,
				}
				if err != nil {
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
