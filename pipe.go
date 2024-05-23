package conc

func idMapperFunc[T any](input T) (output T, skip bool, stop bool) {
	output = input
	return
}

// Mappers connect an input and output channel applying transforms between them
type Mapper[I any, O any] struct {
	RunnerBase[string]
	input  <-chan I
	output chan<- O

	// MapFunc is applied to each value in the input channel
	// and returns a tuple of 3 things - outval, skip, stop
	// if skip is false, outval is sent to the output channel
	// if stop is true, then the entire mapper stops processing any further elements.
	// This mechanism can be used inaddition to the Stop method if sequencing this
	// within the elements of input channel is required
	MapFunc func(I) (O, bool, bool)
	OnDone  func(p *Mapper[I, O])
}

// Creates a new mapper between an input and output channel.
// The ownership of the channels is by the caller and not the Mapper.  Hence they
// will nto be called when the mapper stops.
func NewMapper[T any, U any](input <-chan T, output chan<- U, mapper func(T) (U, bool, bool)) *Mapper[T, U] {
	out := &Mapper[T, U]{
		RunnerBase: NewRunnerBase("stop"),
		input:      input,
		output:     output,
		MapFunc:    mapper,
	}
	out.start()
	return out
}

func (m *Mapper[I, O]) start() {
	m.RunnerBase.start()
	go func() {
		defer m.cleanup()
		for {
			select {
			case <-m.controlChan:
				// stopped - only "stop" allowed here
				return
			case value, ok := <-m.input:
				if ok {
					outval, filter, stop := m.MapFunc(value)
					if !filter {
						m.output <- outval
					}
					if stop {
						return
					}
				} else {
					// we can quit here as there are no more inputs
					return
				}
				break
			}
		}
	}()
}

func NewPipe[T any](input <-chan T, output chan<- T) *Mapper[T, T] {
	return NewMapper(input, output, idMapperFunc)
}
