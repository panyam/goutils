package conc

type Pipe[T any, U any] struct {
	OnClose     func()
	input       <-chan T
	output      chan<- U
	stoppedChan chan bool
	mapper      func(T) U
}

func NewIDPipe[T any](input <-chan T, output chan<- T, mapper func(T) T) *Pipe[T, T] {
	if mapper == nil {
		mapper = IDFunc[T]
	}
	out := NewPipe(input, output, mapper)
	return out
}

func NewPipe[T any, U any](input <-chan T, output chan<- U, mapper func(T) U) *Pipe[T, U] {
	out := &Pipe[T, U]{
		stoppedChan: make(chan bool),
		input:       input,
		output:      output,
		mapper:      mapper,
	}
	out.start()
	return out
}

func (p *Pipe[T, U]) Stop() {
	p.stoppedChan <- true
}

func (p *Pipe[T, U]) start() {
	go func() {
		defer func() {
			close(p.stoppedChan)
			p.stoppedChan = nil
			if p.OnClose != nil {
				p.OnClose()
			}
		}()
		for {
			select {
			case <-p.stoppedChan:
				// stopped - only "stop" allowed here
				return
			case value := <-p.input:
				p.output <- p.mapper(value)
				break
			}
		}
	}()
}
