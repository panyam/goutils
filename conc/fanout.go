package conc

import (
	"sync"
)

type FanOutCmd[T any] struct {
	Name           string
	AddedChannel   chan T
	RemovedChannel <-chan T
}

/**
 * FanOut takes a message from one chanel, applies a mapper function
 * and fans it out to N output channels.
 */
type FanOut[T any, U any] struct {
	Mapper      func(inputs T) U
	selfOwnIn   bool
	inputChan   chan T
	outputChans []chan U
	cmdChan     chan FanOutCmd[U]
	wg          sync.WaitGroup
	isRunning   bool
}

/**
 * Creates a new IDFanout bridge.
 */
func NewIDFanOut[T any](input chan T, mapper func(T) T) *FanOut[T, T] {
	if mapper == nil {
		mapper = IDFunc[T]
	}
	return NewFanOut[T, T](input, mapper)
}

func NewFanOut[T any, U any](inputChan chan T, mapper func(T) U) *FanOut[T, U] {
	selfOwnIn := false
	if inputChan == nil {
		selfOwnIn = true
		inputChan = make(chan T)
	}
	out := &FanOut[T, U]{
		cmdChan:   make(chan FanOutCmd[U], 1),
		inputChan: inputChan,
		selfOwnIn: selfOwnIn,
		Mapper:    mapper,
	}
	out.start()
	return out
}

func (fo *FanOut[T, U]) IsRunning() bool {
	return fo.isRunning
}

func (fo *FanOut[T, U]) InputChannel() chan<- T {
	return fo.inputChan
}

func (fo *FanOut[T, U]) Send(value T) {
	fo.inputChan <- value
}

func (fo *FanOut[T, U]) New() <-chan U {
	output := make(chan U, 1)
	fo.cmdChan <- FanOutCmd[U]{Name: "add", AddedChannel: output}
	return output
}

func (fo *FanOut[T, U]) Remove(output <-chan U) {
	fo.cmdChan <- FanOutCmd[U]{Name: "remove", RemovedChannel: output}
}

func (fo *FanOut[T, U]) Stop() {
	fo.cmdChan <- FanOutCmd[U]{Name: "stop"}
	fo.wg.Wait()
}

func (fo *FanOut[T, U]) start() {
	fo.wg.Add(1)
	fo.isRunning = true
	go func() {
		// keep reading from input and send to outputs
		defer func() {
			if fo.selfOwnIn {
				close(fo.inputChan)
				fo.inputChan = nil
			}
			fo.isRunning = false
			fo.wg.Done()
		}()
		for {
			select {
			case event := <-fo.inputChan:
				result := fo.Mapper(event)
				for _, outputChan := range fo.outputChans {
					outputChan <- result
				}
				break
			case cmd := <-fo.cmdChan:
				if cmd.Name == "stop" {
					return
				} else if cmd.Name == "add" {
					// Add a new reader to our list
					fo.outputChans = append(fo.outputChans, cmd.AddedChannel)
				} else if cmd.Name == "remove" {
					// Remove an existing reader from our list
					for index, ch := range fo.outputChans {
						if ch == cmd.RemovedChannel {
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
