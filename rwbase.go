package conc

import (
	"sync"
	"time"
)

type ReaderWriterBase[T any, D any] struct {
	Info D
	// Time allowed to read the next pong message from the peer.
	WaitTime       time.Duration
	controlChannel chan string
	wg             sync.WaitGroup
}

/**
 * Waits until the socket connection is disconnected or manually stopped.
 */
func (rwb *ReaderWriterBase[T, D]) WaitForFinish() {
	rwb.wg.Wait()
}

func (rwb *ReaderWriterBase[T, D]) start() error {
	rwb.controlChannel = make(chan string)
	rwb.wg.Add(1)
	return nil
}

func (rwb *ReaderWriterBase[T, D]) cleanup() {
	close(rwb.controlChannel)
	rwb.controlChannel = nil
	rwb.wg.Done()
}
