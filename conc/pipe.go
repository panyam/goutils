package conc

type Pipe[T any] struct {
	RunnerBase[string]
	input  <-chan T
	output chan<- T
	OnDone func(p *Pipe[T])
}

func NewPipe[T any](input <-chan T, output chan<- T) *Pipe[T] {
	out := &Pipe[T]{
		RunnerBase: NewRunnerBase("stop"),
		input:      input,
		output:     output,
	}
	out.start()
	return out
}

func (p *Pipe[T]) start() {
	p.RunnerBase.start()
	go func() {
		defer p.cleanup()
		for {
			select {
			case <-p.controlChan:
				// stopped - only "stop" allowed here
				return
			case value, ok := <-p.input:
				if ok {
					p.output <- value
				} else {
					// we can quit here as there are no more inputs
					return
				}
				break
			}
		}
	}()
}
