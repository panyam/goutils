package conc

import (
	"log"
	"time"
)

type FanOut[T any] struct {
	FlushPeriod   time.Duration
	pendingEvents []T
	inputChan     chan T
	outputChans   []chan []T
	cmdChan       chan FanCmd[[]T]
}

func NewFanOut[T any]() *FanOut[T] {
	out := &FanOut[T]{
		FlushPeriod: 100 * time.Millisecond,
		cmdChan:     make(chan FanCmd[[]T]),
	}
	return out
}

func (fo *FanOut[T]) Send(value T) {
	fo.inputChan <- value
}

func (fo *FanOut[T]) NewOutput() (output chan []T) {
	output = make(chan []T)
	fo.AddOutputs(output)
	return
}

func (fo *FanOut[T]) AddOutputs(channels ...chan []T) {
	for _, output := range channels {
		fo.cmdChan <- FanCmd[[]T]{Name: "add", Channel: output}
	}
}

func (fo *FanOut[T]) RemoveOutput(output chan []T) {
	fo.cmdChan <- FanCmd[[]T]{Name: "remove", Channel: output}
}

func (fo *FanOut[T]) Start(output chan []T) {
	if fo.inputChan != nil {
		panic("Input channel is not nil - this needs to be closed first")
	}
	fo.inputChan = make(chan T)
	defer func() {
		close(fo.inputChan)
		fo.inputChan = nil
	}()

	ticker := time.NewTicker(fo.FlushPeriod)
	defer ticker.Stop()

	go func() {
		// keep reading from input and send to outputs
		for {
			select {
			case event := <-fo.inputChan:
				fo.pendingEvents = append(fo.pendingEvents, event)
				break
			case <-ticker.C:
				// Flush
				fo.flush()
				break
			case cmd := <-fo.cmdChan:
				if cmd.Name == "stop" {
					return
				} else if cmd.Name == "add" {
					// Add a new reader to our list
					fo.outputChans = append(fo.outputChans, cmd.Channel)
				} else if cmd.Name == "delete" {
					// Remove an existing reader from our list
					for index, ch := range fo.outputChans {
						if ch == output {
							fo.outputChans[index] = fo.outputChans[len(fo.outputChans)-1]
							fo.outputChans = fo.outputChans[:len(fo.outputChans)-1]
							return
						}
					}
				}
				break
			}
		}
	}()
}

func (fo *FanOut[T]) flush() {
	if len(fo.pendingEvents) == 0 {
		return
	}
	pendingEvents := fo.pendingEvents
	fo.pendingEvents = nil

	// Now blast it out - in another goroutine?
	log.Printf("Flushing %d messages to %d consumers", len(pendingEvents), len(fo.outputChans))
	for _, outputChan := range fo.outputChans {
		outputChan <- pendingEvents
	}
}
