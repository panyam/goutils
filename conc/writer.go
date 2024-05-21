package conc

import (
	"log"
)

// Type of the writer method used by the writer goroutine primitive to serialize its writes.
type WriterFunc[W any] func(W) error

// The typed Writer goroutine type which calls the Write method when it serializes it writes.
type Writer[W any] struct {
	RunnerBase[string]
	msgChannel chan W
	Write      WriterFunc[W]
}

// Creates a new writer instance.   Just like time.Ticker, this initializer
// also starts the Writer loop.   It is upto the caller to Stop this writer when
// done with.  Not doing so can risk the writer to run indefinitely.
func NewWriter[W any](write WriterFunc[W]) *Writer[W] {
	out := Writer[W]{
		RunnerBase: NewRunnerBase("stop"),
		Write:      write,
		msgChannel: make(chan W),
	}
	out.start()
	return &out
}

func (w *Writer[W]) DebugInfo() any {
	return map[string]any{
		"base":    w.RunnerBase.DebugInfo(),
		"msgChan": w.msgChannel,
	}
}

func (ch *Writer[T]) cleanup() {
	log.Println("Cleaning up writer...")
	v := ch.msgChannel
	defer log.Println("Finished cleaning up writer: ", v)
	close(ch.msgChannel)
	ch.msgChannel = nil
	ch.RunnerBase.cleanup()
}

// Returns the channel on which messages can be sent to the Writer.
func (wc *Writer[W]) SendChan() chan W {
	if !wc.IsRunning() {
		return nil
	} else {
		return wc.msgChannel
	}
}

// Sends a message to the Writer.  This is a shortcut for sending
// a message to the underlying channel.
func (wc *Writer[W]) Send(req W) bool {
	if !wc.IsRunning() || wc.msgChannel == nil {
		return false
	}
	wc.msgChannel <- req
	return true
}

// Start writer goroutine
func (wc *Writer[W]) start() {
	wc.RunnerBase.start()
	go func() {
		defer wc.cleanup()
		// ticker := time.NewTicker((wc.WaitTime * 9) / 10)
		// defer ticker.Stop()
		for {
			select {
			case newRequest := <-wc.msgChannel:
				// Here we send a request to the server
				// log.Println("Got a request to write: ", wc.SendChan(), newRequest)
				err := wc.Write(newRequest)
				// log.Println("Handled request to write: ", wc.SendChan(), newRequest)
				if err != nil {
					log.Println("Write Error: ", err)
					return
				}
				break
			case controlRequest := <-wc.controlChan:
				// For now only a "kill" can be sent here
				log.Println("Received kill signal.  Quitting Writer.", controlRequest, wc.SendChan())
				return
			}
		}
	}()
}
