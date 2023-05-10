package conc

import (
	"log"
)

type WriterFunc[W any] func(W) error

type Writer[W any] struct {
	RunnerBase[string]
	msgChannel chan W
	Write      WriterFunc[W]
}

func NewWriter[W any](write WriterFunc[W]) *Writer[W] {
	out := Writer[W]{
		RunnerBase: NewRunnerBase[string]("stop"),
		Write:      write,
		msgChannel: make(chan W),
	}
	out.start()
	return &out
}

func (ch *Writer[T]) cleanup() {
	close(ch.msgChannel)
	ch.msgChannel = nil
	ch.RunnerBase.cleanup()
}

func (wc *Writer[W]) SendChan() chan W {
	if !wc.IsRunning() {
		return nil
	} else {
		return wc.msgChannel
	}
}

func (wc *Writer[W]) Send(req W) bool {
	if !wc.IsRunning() {
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
				err := wc.Write(newRequest)
				if err != nil {
					return
				}
				break
			case controlRequest := <-wc.controlChan:
				// For now only a "kill" can be sent here
				log.Println("Received kill signal.  Quitting Reader.", controlRequest)
				return
			}
		}
	}()
}
