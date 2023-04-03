package conc

import (
	"errors"
	"log"
	"time"
)

type ReaderChan[R any, D any] struct {
	ReaderWriterBase[R, D]
	msgChannel chan ValueOrError[R]
	Read       func() (R, error)
}

func NewReader[R any, D any](read func() (R, error)) *ReaderChan[R, D] {
	out := ReaderChan[R, D]{
		ReaderWriterBase: ReaderWriterBase[R, D]{
			WaitTime: 10 * time.Second,
		},
		Read: read,
	}
	out.start()
	return &out
}

func (ch *ReaderChan[T, D]) cleanup() {
	close(ch.msgChannel)
	ch.msgChannel = nil
	ch.ReaderWriterBase.cleanup()
}

/**
 * Returns whether the connection reader/writer loops are running.
 */
func (ch *ReaderChan[R, D]) IsRunning() bool {
	return ch.msgChannel != nil
}

/**
 * Returns the conn's reader channel.
 */
func (rc *ReaderChan[R, D]) ResultChannel() chan ValueOrError[R] {
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

func (rc *ReaderChan[R, D]) start() (err error) {
	if rc.IsRunning() {
		return errors.New("Channel already running")
	}
	rc.msgChannel = make(chan ValueOrError[R])
	rc.ReaderWriterBase.start()
	go func() {
		defer rc.ReaderWriterBase.cleanup()
		for {
			newMessage, err := rc.Read()
			rc.msgChannel <- ValueOrError[R]{
				Value: newMessage,
				Error: err,
			}
			if err != nil {
				log.Println("Error reading message: ", err)
				break
			}
		}
	}()
	return nil
}
