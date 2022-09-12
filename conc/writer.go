package conc

import (
	"errors"
	"log"
	"time"
)

type WriterChan[W any, D any] struct {
	ReaderWriterBase[W, D]
	msgChannel chan W
	Write      func(msg W) error
	OnClose    func()
}

func (ch *WriterChan[T, D]) cleanup() {
	close(ch.msgChannel)
	ch.msgChannel = nil
	ch.ReaderWriterBase.cleanup()
}

func (wc *WriterChan[W, D]) Send(req W) bool {
	if !wc.IsRunning() {
		log.Println("Connection is not running")
		return false
	}
	wc.msgChannel <- req
	return true
}

func NewWriter[W any, D any](write func(msg W) error) *WriterChan[W, D] {
	out := WriterChan[W, D]{
		ReaderWriterBase: ReaderWriterBase[W, D]{
			WaitTime: 10 * time.Second,
		},
		Write: write,
	}
	out.start()
	return &out
}

/**
 * Returns whether the connection reader/writer loops are running.
 */
func (ch *WriterChan[W, D]) IsRunning() bool {
	return ch.msgChannel != nil
}

// Start writer goroutine
func (wc *WriterChan[W, D]) start() (err error) {
	if wc.IsRunning() {
		return errors.New("Channel already running")
	}
	wc.msgChannel = make(chan W)
	wc.ReaderWriterBase.start()
	log.Println("Starting writer: ")
	go func() {
		ticker := time.NewTicker((wc.WaitTime * 9) / 10)
		defer func() {
			ticker.Stop()
			wc.ReaderWriterBase.cleanup()
			if wc.OnClose != nil {
				wc.OnClose()
			}
		}()

		for {
			select {
			case newRequest := <-wc.msgChannel:
				// Here we send a request to the server
				err := wc.Write(newRequest)
				if err != nil {
					log.Println("Error sending request: ", newRequest, err)
					return
				}
				log.Println("Successfully Sent Request: ", newRequest)
				break
			case controlRequest := <-wc.controlChannel:
				// For now only a "kill" can be sent here
				log.Println("Received kill signal.  Quitting Reader.", controlRequest)
				return
			case <-ticker.C:
				// TODO - use to send pings
			}
		}
	}()
	return
}

/**
 * This method is called to stop the channel
 * If already connected then nothing is done and nil
 * is not already connected, a connection will first be established
 * (including auth and refreshing tokens) and then the reader and
 * writers are started.   SendRequest can be called to send requests
 * to the peer and the (user provided) msgChannel will be used to
 * handle messages from the server.
 */
func (ch *WriterChan[T, D]) Stop() error {
	if !ch.IsRunning() {
		// already running do nothing
		return nil
	}
	ch.controlChannel <- "stop"
	return nil
}
