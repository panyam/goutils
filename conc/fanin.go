package conc

import (
	"sync"
)

type FanInCmd[T any] struct {
	Name           string
	AddedChannel   <-chan T
	RemovedChannel <-chan T
}

type FanIn[T any] struct {
	inChans    []chan T
	selfOwnOut bool
	outChan    chan T
	cmdChan    chan FanInCmd[T]
	pipes      []*Pipe[T, T]
	wg         sync.WaitGroup
	isRunning  bool
}

func NewFanIn[T any](outChan chan T) *FanIn[T] {
	selfOwnOut := false
	if outChan == nil {
		outChan = make(chan T)
		selfOwnOut = true
	}
	out := &FanIn[T]{
		cmdChan:    make(chan FanInCmd[T]),
		outChan:    outChan,
		selfOwnOut: selfOwnOut,
	}
	out.start()
	return out
}

func (fi *FanIn[T]) Channel() chan T {
	return fi.outChan
}

func (fi *FanIn[T]) IsRunning() bool {
	return fi.isRunning
}

func (fi *FanIn[T]) Add(inputs ...<-chan T) {
	for _, input := range inputs {
		fi.cmdChan <- FanInCmd[T]{Name: "add", AddedChannel: input}
	}
}

func (fi *FanIn[T]) Remove(target <-chan T) {
	fi.cmdChan <- FanInCmd[T]{Name: "remove", RemovedChannel: target}
}

func (fi *FanIn[T]) Stop() {
	fi.cmdChan <- FanInCmd[T]{Name: "stop"}
	fi.wg.Wait()
}

func (fi *FanIn[T]) start() {
	fi.wg.Add(1)
	fi.isRunning = true
	go func() {
		defer func() {
			close(fi.cmdChan)
			fi.pipes = nil
			fi.inChans = nil
			fi.cmdChan = nil
			if fi.selfOwnOut {
				close(fi.outChan)
				fi.outChan = nil
			}
			fi.isRunning = false
			fi.wg.Done()
		}()

		for {
			cmd := <-fi.cmdChan
			if cmd.Name == "stop" {
				for _, pipe := range fi.pipes {
					pipe.Stop()
					fi.wg.Done()
				}
				return
			} else if cmd.Name == "add" {
				// Add a new reader to our list
				fi.wg.Add(1)
				pipe := NewPipe(cmd.AddedChannel, fi.outChan, func(x T) T { return x })
				fi.pipes = append(fi.pipes, pipe)
			} else if cmd.Name == "remove" {
				// Remove an existing reader from our list
				for index, ch := range fi.inChans {
					if ch == cmd.RemovedChannel {
						fi.pipes[index].Stop()
						fi.pipes[index] = fi.pipes[len(fi.pipes)-1]
						fi.pipes = fi.pipes[:len(fi.pipes)-1]
						fi.inChans[index] = fi.inChans[len(fi.inChans)-1]
						fi.inChans = fi.inChans[:len(fi.inChans)-1]
						fi.wg.Done()
						break
					}
				}
			}
		}
	}()
}
