package conc

import (
	"log"
	"sync"
	"time"
)

// Reducer is a way to collect messages of type T in some kind of window
// and reduce them to type U.  For example this could be used to batch messages
// into a list every 10 seconds.  Alternatively if a time based window is not
// used a reduction can be invokved manually.
type Reducer[T any, U any] struct {
	FlushPeriod   time.Duration
	ReduceFunc    func(inputs []T) (outputs U)
	pendingEvents []T
	selfOwnIn     bool
	inputChan     chan T
	selfOwnOut    bool
	outputChan    chan U
	cmdChan       chan reducerCmd[U]
	wg            sync.WaitGroup
}

type reducerCmd[T any] struct {
	Name    string
	Channel chan T
}

// A Reducer that simply collects events of type T into a list (of type []T)
func NewIDReducer[T any](inputChan chan T, outputChan chan []T) *Reducer[T, []T] {
	out := NewReducer(inputChan, outputChan)
	out.ReduceFunc = IDFunc[[]T]
	return out
}

// The reducer over generic input and output types.  The input channel
// can be provided on which the reducer will read messages.  If an input
// channel is not provided then the reducer will create one (and own its
// lifecycle).
// Just like other runners, the Reducer starts as soon as it is created.
func NewReducer[T any, U any](inputChan chan T, outputChan chan U) *Reducer[T, U] {
	selfOwnIn := false
	if inputChan == nil {
		selfOwnIn = true
		inputChan = make(chan T)
	}
	selfOwnOut := false
	if outputChan == nil {
		selfOwnOut = true
		outputChan = make(chan U)
	}
	out := &Reducer[T, U]{
		FlushPeriod: 100 * time.Millisecond,
		cmdChan:     make(chan reducerCmd[U]),
		inputChan:   inputChan,
		selfOwnIn:   selfOwnIn,
		outputChan:  outputChan,
		selfOwnOut:  selfOwnOut,
	}
	out.start()
	return out
}

// The channel onto which messages can be sent (to be reduced)
func (fo *Reducer[T, U]) SendChan() chan<- T {
	return fo.inputChan
}

// Send a mesasge/value onto this reducer for (eventual) reduction.
func (fo *Reducer[T, U]) Send(value T) {
	fo.inputChan <- value
}

// Stops the reducer and closes all channels it owns.
func (fo *Reducer[T, U]) Stop() {
	fo.cmdChan <- reducerCmd[U]{Name: "stop"}
	fo.wg.Wait()
}

func (fo *Reducer[T, U]) start() {
	ticker := time.NewTicker(fo.FlushPeriod)
	fo.wg.Add(1)
	go func() {
		// keep reading from input and send to outputs
		defer func() {
			defer ticker.Stop()
			if fo.selfOwnIn {
				close(fo.inputChan)
				fo.inputChan = nil
			}
			fo.wg.Done()
		}()
		for {
			select {
			case event := <-fo.inputChan:
				fo.pendingEvents = append(fo.pendingEvents, event)
				break
			case <-ticker.C:
				// Flush
				fo.Flush()
				break
			case cmd := <-fo.cmdChan:
				if cmd.Name == "stop" {
					return
				}
				break
			}
		}
	}()
}

func (fo *Reducer[T, U]) Flush() {
	if len(fo.pendingEvents) > 0 {
		log.Printf("Flushing %d messages.", len(fo.pendingEvents))
		joinedEvents := fo.ReduceFunc(fo.pendingEvents)
		fo.pendingEvents = nil
		fo.outputChan <- joinedEvents
	}
}
