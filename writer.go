package conc

import (
	"errors"
	"log"
	"time"
)

type Writer[W any] struct {
	ReaderWriterBase[W]
	msgChannel chan W
	Write      func(msg W) error
	OnClose    func()
}

func NewWriter[W any](write func(msg W) error) *Writer[W] {
	out := Writer[W]{
		ReaderWriterBase: ReaderWriterBase[W]{
			WaitTime: 10 * time.Second,
		},
		Write: write,
	}
	out.start()
	return &out
}

func (ch *Writer[T]) cleanup() {
	close(ch.msgChannel)
	ch.msgChannel = nil
	ch.ReaderWriterBase.cleanup()
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
		log.Println("Connection is not running")
		return false
	}
	wc.msgChannel <- req
	return true
}

/**
 * Returns whether the connection reader/writer loops are running.
 */
func (ch *Writer[W]) IsRunning() bool {
	return ch.msgChannel != nil
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
func (ch *Writer[T]) Stop() error {
	if !ch.IsRunning() && ch.controlChannel != nil {
		// already running do nothing
		return nil
	}
	ch.controlChannel <- "stop"
	return nil
}

// Start writer goroutine
func (wc *Writer[W]) start() (err error) {
	if wc.IsRunning() {
		return errors.New("Channel already running")
	}
	wc.msgChannel = make(chan W)
	wc.ReaderWriterBase.start()
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
					return
				}
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
