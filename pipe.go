package conc

type Pipe[T any, U any] struct {
	OnClose     func(p *Pipe[T, U])
	input       <-chan T
	output      chan<- U
	controlChan chan bool
	mapper      func(T) U
	isRunning   bool
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
		controlChan: make(chan bool),
		input:       input,
		output:      output,
		mapper:      mapper,
	}
	out.start()
	return out
}

func (p *Pipe[T, U]) InChan() <-chan T {
	return p.input
}

func (p *Pipe[T, U]) OutChan() chan<- U {
	return p.output
}

func (p *Pipe[T, U]) Stop() {
	p.controlChan <- true
}

func (p *Pipe[T, U]) IsRunning() bool {
	return p.isRunning
}

func (p *Pipe[T, U]) start() {
	p.isRunning = true
	go func() {
		defer func() {
			if p.OnClose != nil {
				p.OnClose(p)
			}
			close(p.controlChan)
			p.controlChan = nil
			p.isRunning = false
		}()
		for {
			select {
			case <-p.controlChan:
				// stopped - only "stop" allowed here
				return
			case value, ok := <-p.input:
				if ok {
					p.output <- p.mapper(value)
				} else {
					// we can quit here as there are no more inputs
					return
				}
				break
			}
		}
	}()
}
