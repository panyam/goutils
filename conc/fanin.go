package conc

import (
	"sync"
)

type FanCmd[T any] struct {
	Name    string
	Channel chan T
}

func FanIn[T any](readers ...chan T) (chan T, chan FanCmd[T]) {
	var wg sync.WaitGroup
	resultChannel := make(chan T)
	controlChannel := make(chan FanCmd[T])

	var cmdChans []chan string
	addChannel := func(reader chan T) {
		wg.Add(1)
		cmdChan := Pipe(reader, resultChannel, func() {
			wg.Done()
		})
		cmdChans = append(cmdChans, cmdChan)
	}

	removeChannel := func(reader chan T) {
		for index, ch := range readers {
			if ch == reader {
				cmdChans[index] <- "stop"
				cmdChans[index] = cmdChans[len(cmdChans)-1]
				cmdChans = cmdChans[:len(cmdChans)-1]
				readers[index] = readers[len(readers)-1]
				readers = readers[:len(readers)-1]
				return
			}
		}
	}

	for _, reader := range readers {
		addChannel(reader)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			cmd := <-controlChannel
			if cmd.Name == "stop" || cmd.Name == "pause" {
				for _, cmdChan := range cmdChans {
					cmdChan <- cmd.Name
				}
				if cmd.Name == "stop" {
					wg.Done()
					return
				}
			} else if cmd.Name == "add" {
				// Add a new reader to our list
				addChannel(cmd.Channel)
			} else if cmd.Name == "delete" {
				// Remove an existing reader from our list
				removeChannel(cmd.Channel)
			}
		}
	}()

	// And a goroutine to wait for all these to be done
	go func() {
		wg.Wait()
		close(controlChannel)
		close(resultChannel)
	}()
	return resultChannel, controlChannel
}
