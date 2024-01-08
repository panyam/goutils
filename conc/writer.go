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

func (wc *Writer[W]) SendChan() chan W {
	if !wc.IsRunning() {
		return nil
	} else {
		return wc.msgChannel
	}
}

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
