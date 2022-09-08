package conc

import (
	"errors"
	"log"
	"sync"
	"time"
)

type ValueOrError[T any] struct {
	Value *T
	Error error
}

type ReaderWriterBase[T any, D any] struct {
	Info D
	// Time allowed to read the next pong message from the peer.
	WaitTime       time.Duration
	controlChannel chan string
	wg             sync.WaitGroup
}

type ReaderChan[R any, D any] struct {
	ReaderWriterBase[R, D]
	msgChannel chan ValueOrError[R]
	Read       func() (*R, error)
}

type WriterChan[W any, D any] struct {
	ReaderWriterBase[W, D]
	msgChannel chan *W
	Write      func(msg *W) error
	OnClose    func()
}

/**
 * Waits until the socket connection is disconnected or manually stopped.
 */
func (rwb *ReaderWriterBase[T, D]) WaitForFinish() {
	rwb.wg.Wait()
}

func (rwb *ReaderWriterBase[T, D]) Start() error {
	rwb.controlChannel = make(chan string)
	rwb.wg.Add(1)
	return nil
}

func (rwb *ReaderWriterBase[T, D]) cleanup() {
	close(rwb.controlChannel)
	rwb.controlChannel = nil
	rwb.wg.Done()
}

func (ch *ReaderChan[T, D]) cleanup() {
	close(ch.msgChannel)
	ch.msgChannel = nil
	ch.ReaderWriterBase.cleanup()
}

func (ch *WriterChan[T, D]) cleanup() {
	close(ch.msgChannel)
	ch.msgChannel = nil
	ch.ReaderWriterBase.cleanup()
}

func (wc *WriterChan[W, D]) Send(req *W) bool {
	if !wc.IsRunning() {
		log.Println("Connection is not running")
		return false
	}
	wc.msgChannel <- req
	return true
}

func NewReader[R any, D any](read func() (*R, error)) *ReaderChan[R, D] {
	out := ReaderChan[R, D]{
		ReaderWriterBase: ReaderWriterBase[R, D]{
			WaitTime: 10 * time.Second,
		},
		Read: read,
	}
	return &out
}

/**
 * Returns whether the connection reader/writer loops are running.
 */
func (ch *ReaderChan[R, D]) IsRunning() bool {
	return ch.msgChannel != nil
}

func NewWriter[W any, D any](write func(msg *W) error) *WriterChan[W, D] {
	out := WriterChan[W, D]{
		ReaderWriterBase: ReaderWriterBase[W, D]{
			WaitTime: 10 * time.Second,
		},
		Write: write,
	}
	return &out
}

/**
 * Returns whether the connection reader/writer loops are running.
 */
func (ch *WriterChan[W, D]) IsRunning() bool {
	return ch.msgChannel != nil
}

/**
 * Returns the conn's reader channel.
 */
func (rc *ReaderChan[R, D]) Channel() chan ValueOrError[R] {
	return rc.msgChannel
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
func (ch *ReaderChan[T, D]) Stop() error {
	if !ch.IsRunning() {
		// already running do nothing
		return nil
	}
	ch.controlChannel <- "stop"
	return nil
}

func (rc *ReaderChan[R, D]) Start() (err error) {
	if rc.IsRunning() {
		return errors.New("Channel already running")
	}
	rc.msgChannel = make(chan ValueOrError[R])
	rc.ReaderWriterBase.Start()
	log.Println("Starting reader: ")
	go func() {
		defer rc.ReaderWriterBase.cleanup()
		for {
			newMessage, err := rc.Read()
			rc.msgChannel <- ValueOrError[R]{
				Value: newMessage,
				Error: err,
			}
			if err != nil || newMessage == nil {
				log.Println("Error reading message: ", err)
				break
			}
		}
	}()
	return nil
}

// Start writer goroutine
func (wc *WriterChan[W, D]) Start() (err error) {
	if wc.IsRunning() {
		return errors.New("Channel already running")
	}
	wc.msgChannel = make(chan *W)
	wc.ReaderWriterBase.Start()
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
