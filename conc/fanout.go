package conc

import (
	"log"
	"time"
)

type FanOut[T any, U any] struct {
	FlushPeriod   time.Duration
	Joiner        func(inputs []T) (outputs U)
	pendingEvents []T
	inputChan     chan T
	outputChans   []chan U
	cmdChan       chan FanCmd[U]
}

func NewFanOut[T any, U any](inputChan chan T) *FanOut[T, U] {
	if inputChan == nil {
		inputChan = make(chan T)
	}
	out := &FanOut[T, U]{
		FlushPeriod: 100 * time.Millisecond,
		cmdChan:     make(chan FanCmd[U]),
		inputChan:   inputChan,
	}
	out.start()
	return out
}

func (fo *FanOut[T, U]) InputChannel() chan<- T {
	return fo.inputChan
}

func (fo *FanOut[T, U]) Send(value T) {
	fo.inputChan <- value
}

func (fo *FanOut[T, U]) New() chan U {
	output := make(chan U)
	fo.cmdChan <- FanCmd[U]{Name: "add", Channel: output}
	return output
}

func (fo *FanOut[T, U]) Remove(output chan U) {
	fo.cmdChan <- FanCmd[U]{Name: "remove", Channel: output}
}

func (fo *FanOut[T, U]) Stop() {
	fo.cmdChan <- FanCmd[U]{Name: "stop"}
}

func (fo *FanOut[T, U]) start() {
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
				} else if cmd.Name == "remove" {
					// Remove an existing reader from our list
					for index, ch := range fo.outputChans {
						if ch == cmd.Channel {
							close(ch)
							fo.outputChans[index] = fo.outputChans[len(fo.outputChans)-1]
							fo.outputChans = fo.outputChans[:len(fo.outputChans)-1]
							break
						}
					}
				}
				break
			}
		}
	}()
}

func (fo *FanOut[T, U]) flush() {
	if len(fo.pendingEvents) == 0 {
		return
	}
	pendingEvents := fo.pendingEvents
	fo.pendingEvents = nil

	// Now blast it out - in another goroutine?
	log.Printf("Flushing %d messages to %d consumers", len(pendingEvents), len(fo.outputChans))
	for _, outputChan := range fo.outputChans {
		outputChan <- fo.Joiner(pendingEvents)
	}
}
