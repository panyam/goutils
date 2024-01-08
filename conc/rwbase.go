package conc

import (
	"errors"
	"sync"
)

type RunnerBase[C any] struct {
	controlChan chan C
	isRunning   bool
	wg          sync.WaitGroup
	stopVal     C
}

func NewRunnerBase[C any](stopVal C) RunnerBase[C] {
	return RunnerBase[C]{
		controlChan: make(chan C, 1),
		stopVal:     stopVal,
	}
}

func (r *RunnerBase[R]) DebugInfo() any {
	return map[string]any{
		"ctrlChan":  r.controlChan,
		"stopVal":   r.stopVal,
		"isRunning": r.isRunning,
	}
}

func (r *RunnerBase[C]) IsRunning() bool {
	return r.isRunning
}

/**
 * This method is intentionally private.  It is to be inherited by child
 * types and then called after their initialization is done.
 */
func (r *RunnerBase[C]) start() error {
	if r.IsRunning() {
		return errors.New("Channel already running")
	}
	r.isRunning = true
	r.wg.Add(1)
	return nil
}

/**
 * This method is called to stop the runner.  It is upto the child classes
 * to listen to messages on the control channel and initiate the wind-down
 * and cleanup process.
 */
func (r *RunnerBase[C]) Stop() error {
	if !r.IsRunning() && r.controlChan != nil {
		// already running do nothing
		return nil
	}
	r.controlChan <- r.stopVal
	r.isRunning = false
	r.wg.Wait()
	return nil
}

func (r *RunnerBase[C]) cleanup() {
	close(r.controlChan)
	r.controlChan = nil
	r.wg.Done()
}
