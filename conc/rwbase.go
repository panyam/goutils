package conc

import (
	"sync"
	"time"
)

type ReaderWriterBase[T any] struct {
	// Time allowed to read the next pong message from the peer.
	WaitTime       time.Duration
	controlChannel chan string
	wg             sync.WaitGroup
}

/**
 * Waits until the socket connection is disconnected or manually stopped.
 */
func (rwb *ReaderWriterBase[T]) WaitForFinish() {
	rwb.wg.Wait()
}

func (rwb *ReaderWriterBase[T]) start() error {
	rwb.controlChannel = make(chan string, 1)
	rwb.wg.Add(1)
	return nil
}

func (rwb *ReaderWriterBase[T]) cleanup() {
	close(rwb.controlChannel)
	rwb.controlChannel = nil
	rwb.wg.Done()
}
