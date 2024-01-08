package conc

import (
	"log"
)

type FilterFunc[T any] func(*T) *T

type FanOutCmd[T any] struct {
	Name           string
	Filter         FilterFunc[T]
	SelfOwned      bool
	AddedChannel   chan<- T
	RemovedChannel chan<- T
	CallbackChan   chan error
}

/**
 * FanOut takes a message from one chanel, applies a mapper function
 * and fans it out to N output channels.
 */
type FanOut[T any] struct {
	RunnerBase[FanOutCmd[T]]
	selfOwnIn       bool
	inputChan       chan T
	outputChans     []chan<- T
	outputSelfOwned []bool
	outputFilters   []FilterFunc[T]
}

func NewFanOut[T any](inputChan chan T) *FanOut[T] {
	selfOwnIn := false
	if inputChan == nil {
		selfOwnIn = true
		inputChan = make(chan T)
	}
	out := &FanOut[T]{
		RunnerBase: NewRunnerBase(FanOutCmd[T]{Name: "stop"}),
		inputChan:  inputChan,
		selfOwnIn:  selfOwnIn,
	}
	out.start()
	return out
}

func (fo *FanOut[T]) DebugInfo() any {
	return map[string]any{
		"inputChan":    fo.inputChan,
		"outputChan":   fo.outputChans,
		"outputChanSO": fo.outputSelfOwned,
	}
}

func (fo *FanOut[T]) Count() int {
	return len(fo.outputChans)
}

func (fo *FanOut[T]) SendChan() <-chan T {
	return fo.inputChan
}

func (fo *FanOut[T]) Send(value T) {
	fo.inputChan <- value
}

func (fo *FanOut[T]) New(filter FilterFunc[T]) chan T {
	output := make(chan T, 1)
	fo.Add(output, filter, false)
	return output
}

/**
 * Adds a new fan out receiver with an optional filter method.
 * The filter method can be used to filter out message to certain
 * listeners if necessary.
 */
func (fo *FanOut[T]) Add(output chan<- T, filter FilterFunc[T], wait bool) (callbackChan chan error) {
	if wait {
		callbackChan = make(chan error, 1)
	}
	fo.controlChan <- FanOutCmd[T]{Name: "add", AddedChannel: output, Filter: filter, CallbackChan: callbackChan}
	return
}

func (fo *FanOut[T]) Remove(output chan<- T, wait bool) (callbackChan chan error) {
	if wait {
		callbackChan = make(chan error)
	}
	fo.controlChan <- FanOutCmd[T]{Name: "remove", RemovedChannel: output, CallbackChan: callbackChan}
	return
}

func (fo *FanOut[T]) cleanup() {
	if fo.selfOwnIn {
		close(fo.inputChan)
	}
	fo.inputChan = nil
	// close any output channels *we* own
	for index, ch := range fo.outputChans {
		if fo.outputSelfOwned[index] && ch != nil {
			close(ch)
		}
	}
	fo.outputChans = nil
	fo.outputFilters = nil
	fo.outputSelfOwned = nil
	fo.RunnerBase.cleanup()
}

func (fo *FanOut[T]) start() {
	fo.RunnerBase.start()

	go func() {
		defer fo.cleanup()

		// keep reading from input and send to outputs
		for {
			select {
			case event := <-fo.inputChan:
				if fo.outputChans != nil {
					for index, outputChan := range fo.outputChans {
						if outputChan != nil {
							if fo.outputFilters[index] != nil {
								newevent := fo.outputFilters[index](&event)
								if newevent != nil {
									outputChan <- *newevent
								}
							} else {
								// log.Println("Sending Event to chan: ", event, outputChan)
								outputChan <- event
								// log.Println("Finished Sending Event to chan: ", event, outputChan)
							}
						}
					}
				}
				break
			case cmd := <-fo.controlChan:
				if cmd.Name == "stop" {
					return
				}

				if cmd.Name == "add" {
					// Add a new reader to our list
					// check for dup?
					found := false
					if fo.outputChans != nil {
						// all good
						for _, oc := range fo.outputChans {
							if oc == cmd.AddedChannel {
								found = true
								// Or should we replace this?
								log.Println("Output Channel already exists.  Will skip.  Remove it first if you want to add again or change filter funcs", cmd.AddedChannel, oc, fo.outputChans)
								break
							}
						}
					}
					if !found {
						fo.outputChans = append(fo.outputChans, cmd.AddedChannel)
						fo.outputSelfOwned = append(fo.outputSelfOwned, cmd.SelfOwned)
						fo.outputFilters = append(fo.outputFilters, cmd.Filter)
					}
					if cmd.CallbackChan != nil {
						cmd.CallbackChan <- nil
					}
				} else if cmd.Name == "remove" {
					// Remove an existing reader from our list
					for index, ch := range fo.outputChans {
						if ch == cmd.RemovedChannel {
							// log.Println("Before Removing channel: ", ch, len(fo.outputChans), fo.outputChans)
							if fo.outputSelfOwned[index] {
								close(ch)
							}
							fo.outputSelfOwned[index] = fo.outputSelfOwned[len(fo.outputSelfOwned)-1]
							fo.outputSelfOwned = fo.outputSelfOwned[:len(fo.outputSelfOwned)-1]

							fo.outputChans[index] = fo.outputChans[len(fo.outputChans)-1]
							fo.outputChans = fo.outputChans[:len(fo.outputChans)-1]

							fo.outputFilters[index] = fo.outputFilters[len(fo.outputFilters)-1]
							fo.outputFilters = fo.outputFilters[:len(fo.outputFilters)-1]
							// log.Println("After Removing channel: ", ch, len(fo.outputChans), fo.outputChans)
							break
						}
					}
					if cmd.CallbackChan != nil {
						cmd.CallbackChan <- nil
					}
				}
				break
			}
		}
	}()
}
