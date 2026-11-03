package coakkahttp

import (
	"errors"
	"runtime"
)

// dispatcher is the one inbound reader, not an affinity-only helper. Core CPU
// scopes are OS-thread-owned; a migrating goroutine cannot safely hold one.
// The unbuffered control channel adds no backlog and is NOT used per event by
// Service: its entire existing pump executes here. Responses remain direct.
type dispatcher struct {
	tasks    chan dispatcherTask
	done     chan struct{}
	cpu      CPUInfo
	closeErr error // published by closing done
}

type dispatcherTask struct {
	run  func()
	done chan struct{}
	stop bool
}

func openDispatcher(config Config) (*dispatcher, *nativeService, error) {
	type opened struct {
		native *nativeService
		err    error
	}
	owner := &dispatcher{tasks: make(chan dispatcherTask), done: make(chan struct{})}
	ready := make(chan opened, 1)
	go func() {
		defer close(owner.done)
		runtime.LockOSThread()
		clean := false
		defer func() {
			// Go terminates an OS thread when its locked goroutine exits. On any
			// uncertain restoration/entry failure, retire it instead of leaking
			// altered affinity into Go's general scheduler. No Go OS policy code.
			if clean {
				runtime.UnlockOSThread()
			}
		}()
		scope, cpu, err := enterNativeCPU(config.CPU)
		if err != nil {
			ready <- opened{err: err}
			return
		}
		owner.cpu = cpu
		native, err := openNativeService(config)
		if err != nil {
			leaveErr := scope.leave()
			clean = leaveErr == nil
			ready <- opened{err: errors.Join(err, leaveErr)}
			return
		}
		ready <- opened{native: native}
		for task := range owner.tasks {
			if task.stop {
				// Caller has successfully destroyed/joined Core, and no pump or
				// acquired native call remains. Refused destroys never get here.
				owner.closeErr = scope.leave()
				clean = owner.closeErr == nil
				return
			}
			task.run()
			if task.done != nil {
				close(task.done)
			}
		}
	}()
	result := <-ready
	if result.err != nil {
		<-owner.done
		return nil, nil, result.err
	}
	return owner, result.native, nil
}

// call is only for startup and the explicit low-level pull API. The Service
// pump calls native directly on this owner, with no per-request command channel.
func (owner *dispatcher) call(run func()) {
	done := make(chan struct{})
	owner.tasks <- dispatcherTask{run: run, done: done}
	<-done
}

func (owner *dispatcher) runPump(run func()) {
	owner.tasks <- dispatcherTask{run: run}
}

func (owner *dispatcher) close() error {
	owner.tasks <- dispatcherTask{stop: true}
	<-owner.done
	return owner.closeErr
}
