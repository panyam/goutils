package conc

import (
	"log"
)

type FilterFunc[T any] func(T) bool

type FanOutCmd[T any, U any] struct {
	Name           string
	Filter         FilterFunc[T]
	SelfOwned      bool
	AddedChannel   chan<- U
	RemovedChannel chan<- U
	CallbackChan   chan error
}

/**
 * FanOut takes a message from one chanel, applies a mapper function
 * and fans it out to N output channels.
 */
type FanOut[T any, U any] struct {
	RunnerBase[FanOutCmd[T, U]]
	Mapper func(inputs T) U

	selfOwnIn       bool
	inputChan       chan T
	outputChans     []chan<- U
	outputSelfOwned []bool
	outputFilters   []FilterFunc[T]
}

/**
 * Creates a new IDFanout bridge.
 */
func NewIDFanOut[T any](input chan T, mapper func(T) T) *FanOut[T, T] {
	if mapper == nil {
		mapper = IDFunc[T]
	}
	return NewFanOut(input, mapper)
}

func NewFanOut[T any, U any](inputChan chan T, mapper func(T) U) *FanOut[T, U] {
	if mapper == nil {
		log.Panic("Mapper MUST be provided to convert between T and U")
	}
	selfOwnIn := false
	if inputChan == nil {
		selfOwnIn = true
		inputChan = make(chan T)
	}
	out := &FanOut[T, U]{
		RunnerBase: NewRunnerBase(FanOutCmd[T, U]{Name: "stop"}),
		inputChan:  inputChan,
		selfOwnIn:  selfOwnIn,
		Mapper:     mapper,
	}
	out.start()
	return out
}

func (fo *FanOut[T, U]) Count() int {
	return len(fo.outputChans)
}

func (fo *FanOut[T, U]) SendChan() <-chan T {
	return fo.inputChan
}

func (fo *FanOut[T, U]) Send(value T) {
	fo.inputChan <- value
}

func (fo *FanOut[T, U]) New(filter func(T) bool) chan U {
	output := make(chan U, 1)
	fo.Add(output, filter)
	return output
}

/**
 * Adds a new fan out receiver with an optional filter method.
 * The filter method can be used to filter out message to certain
 * listeners if necessary.
 */
func (fo *FanOut[T, U]) Add(output chan<- U, filter func(T) bool) {
	fo.controlChan <- FanOutCmd[T, U]{Name: "add", AddedChannel: output, Filter: filter}
}

func (fo *FanOut[T, U]) Remove(output chan<- U, callbackChan chan error) {
	fo.controlChan <- FanOutCmd[T, U]{Name: "remove", RemovedChannel: output, CallbackChan: callbackChan}
}

func (fo *FanOut[T, U]) cleanup() {
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

func (fo *FanOut[T, U]) start() {
	fo.RunnerBase.start()

	go func() {
		defer fo.cleanup()

		// keep reading from input and send to outputs
		for {
			select {
			case event := <-fo.inputChan:
				result := fo.Mapper(event)
				if fo.outputChans != nil {
					for index, outputChan := range fo.outputChans {
						if outputChan != nil && fo.outputFilters[index] == nil || fo.outputFilters[index](event) {
							outputChan <- result
						}
					}
				}
				break
			case cmd := <-fo.controlChan:
				if cmd.Name == "stop" {
					return
				} else if cmd.Name == "add" {
					// Add a new reader to our list
					// check for dup?
					found := false
					if fo.outputChans != nil {
						// all good
						for _, oc := range fo.outputChans {
							if oc == cmd.AddedChannel {
								found = true
								// Or should we replace this?
								log.Println("Output Channel already exists.  Will skip.  Remove it first if you want to add again or change filter funcs")
								break
							}
						}
					}
					if !found {
						fo.outputChans = append(fo.outputChans, cmd.AddedChannel)
						fo.outputSelfOwned = append(fo.outputSelfOwned, cmd.SelfOwned)
						fo.outputFilters = append(fo.outputFilters, cmd.Filter)
					}
				} else if cmd.Name == "remove" {
					// Remove an existing reader from our list
					for index, ch := range fo.outputChans {
						if ch == cmd.RemovedChannel {
							log.Println("Before Removing channel: ", ch, len(fo.outputChans), fo.outputChans)
							if fo.outputSelfOwned[index] {
								close(ch)
							}
							fo.outputSelfOwned[index] = fo.outputSelfOwned[len(fo.outputSelfOwned)-1]
							fo.outputSelfOwned = fo.outputSelfOwned[:len(fo.outputSelfOwned)-1]

							fo.outputChans[index] = fo.outputChans[len(fo.outputChans)-1]
							fo.outputChans = fo.outputChans[:len(fo.outputChans)-1]

							fo.outputFilters[index] = fo.outputFilters[len(fo.outputFilters)-1]
							fo.outputFilters = fo.outputFilters[:len(fo.outputFilters)-1]
							log.Println("After Removing channel: ", ch, len(fo.outputChans), fo.outputChans)
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
