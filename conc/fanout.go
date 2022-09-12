package conc

import (
	"sync"
)

type FanOut[T any, U any] struct {
	Mapper      func(inputs T) U
	selfOwnIn   bool
	inputChan   chan T
	outputChans []chan U
	cmdChan     chan FanCmd[U]
	wg          sync.WaitGroup
	isRunning   bool
}

func NewIDFanOut[T any](input chan T, mapper func(T) T) *FanOut[T, T] {
	if mapper == nil {
		mapper = IDFunc[T]
	}
	return NewFanOut[T, T](input)
}

func NewFanOut[T any, U any](inputChan chan T) *FanOut[T, U] {
	selfOwnIn := false
	if inputChan == nil {
		selfOwnIn = true
		inputChan = make(chan T)
	}
	out := &FanOut[T, U]{
		cmdChan:   make(chan FanCmd[U], 1),
		inputChan: inputChan,
		selfOwnIn: selfOwnIn,
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

func (fo *FanOut[T, U]) New() chan U {
	output := make(chan U, 1)
	fo.cmdChan <- FanCmd[U]{Name: "add", Channel: output}
	return output
}

func (fo *FanOut[T, U]) Remove(output chan U) {
	fo.cmdChan <- FanCmd[U]{Name: "remove", Channel: output}
}

func (fo *FanOut[T, U]) Stop() {
	fo.cmdChan <- FanCmd[U]{Name: "stop"}
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
				for _, outputChan := range fo.outputChans {
					if fo.Mapper != nil {
						outputChan <- fo.Mapper(event)
					}
				}
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
