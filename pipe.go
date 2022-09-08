package conc

func Pipe[T any](input chan T, output chan T, onDone func()) (cmdChan chan string) {
	cmdChan = make(chan string)
	go func() {
		defer close(cmdChan)
		if onDone != nil {
			defer onDone()
		}
		state := "running"
		for {
			cmd := ""
			if state == "running" {
				select {
				case cmd = <-cmdChan:
					break
				case value := <-input:
					output <- value
					break
				}
			} else {
				cmd = <-cmdChan
			}

			if cmd == "stop" {
				return
			} else if cmd == "pause" {
				state = "paused"
			}
		}
	}()
	return cmdChan
}
