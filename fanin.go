package conc

import (
	"sync"
)

type FanCmd[T any] struct {
	Name    string
	Channel chan T
}

type FanIn[T any] struct {
	inChans []chan T
	outChan chan T
	cmdChan chan FanCmd[T]
	pipes   []*Pipe[T, T]
	wg      sync.WaitGroup
}

func NewFanIn[T any]() *FanIn[T] {
	out := &FanIn[T]{
		cmdChan: make(chan FanCmd[T]),
		outChan: make(chan T),
	}
	out.start()
	return out
}

func (fi *FanIn[T]) Channel() chan T {
	return fi.outChan
}

func (fi *FanIn[T]) Add(inputs ...chan T) {
	for _, input := range inputs {
		fi.cmdChan <- FanCmd[T]{Name: "add", Channel: input}
	}
}

func (fi *FanIn[T]) Remove(target chan T) {
	fi.cmdChan <- FanCmd[T]{Name: "remove", Channel: target}
}

func (fi *FanIn[T]) Stop() {
	fi.cmdChan <- FanCmd[T]{Name: "stop"}
}

func (fi *FanIn[T]) start() *FanIn[T] {
	fi.wg.Add(1)
	go func() {
		defer fi.wg.Done()
		for {
			cmd := <-fi.cmdChan
			if cmd.Name == "stop" {
				for _, pipe := range fi.pipes {
					pipe.Stop()
				}
				fi.wg.Done()
				return
			} else if cmd.Name == "add" {
				// Add a new reader to our list
				fi.wg.Add(1)
				pipe := NewPipe(cmd.Channel, fi.outChan, func(x T) T { return x })
				fi.pipes = append(fi.pipes, pipe)
			} else if cmd.Name == "remove" {
				// Remove an existing reader from our list
				for index, ch := range fi.inChans {
					if ch == cmd.Channel {
						fi.pipes[index].Stop()
						fi.pipes[index] = fi.pipes[len(fi.pipes)-1]
						fi.pipes = fi.pipes[:len(fi.pipes)-1]
						fi.inChans[index] = fi.inChans[len(fi.inChans)-1]
						fi.inChans = fi.inChans[:len(fi.inChans)-1]
						break
					}
				}
			}
		}
	}()

	// And a goroutine to wait for all these to be done
	go func() {
		fi.wg.Wait()
		close(fi.cmdChan)
		close(fi.outChan)
		fi.cmdChan = nil
		fi.outChan = nil
	}()
	return fi
}
