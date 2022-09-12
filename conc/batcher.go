package conc

import (
	"log"
	"sync"
	"time"
)

type Batcher[T any, U any] struct {
	FlushPeriod   time.Duration
	Joiner        func(inputs []T) (outputs U)
	pendingEvents []T
	selfOwnIn     bool
	inputChan     chan T
	outputChan    chan U
	cmdChan       chan FanCmd[U]
	wg            sync.WaitGroup
}

func NewBatcher[T any, U any](inputChan chan T) *Batcher[T, U] {
	selfOwnIn := false
	if inputChan == nil {
		selfOwnIn = true
		inputChan = make(chan T)
	}
	out := &Batcher[T, U]{
		FlushPeriod: 100 * time.Millisecond,
		cmdChan:     make(chan FanCmd[U]),
		inputChan:   inputChan,
		selfOwnIn:   selfOwnIn,
	}
	out.start()
	return out
}

func (fo *Batcher[T, U]) InputChannel() chan<- T {
	return fo.inputChan
}

func (fo *Batcher[T, U]) Send(value T) {
	fo.inputChan <- value
}

func (fo *Batcher[T, U]) New() chan U {
	output := make(chan U)
	fo.cmdChan <- FanCmd[U]{Name: "add", Channel: output}
	return output
}

func (fo *Batcher[T, U]) Remove(output chan U) {
	fo.cmdChan <- FanCmd[U]{Name: "remove", Channel: output}
}

func (fo *Batcher[T, U]) Stop() {
	fo.cmdChan <- FanCmd[U]{Name: "stop"}
	fo.wg.Wait()
}

func (fo *Batcher[T, U]) start() {
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
				fo.flush()
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

func (fo *Batcher[T, U]) flush() {
	if len(fo.pendingEvents) > 0 {
		log.Printf("Flushing %d messages.", len(fo.pendingEvents))
		joinedEvents := fo.Joiner(fo.pendingEvents)
		fo.pendingEvents = nil
		fo.outputChan <- joinedEvents
	}
}
