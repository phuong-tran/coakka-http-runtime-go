package coakkahttp

import (
	"errors"
	"fmt"
	"time"
)

// DrainState is Core's completion observation, independent of event leases.
// Unknown future values are preserved; they never imply graceful completion.
type DrainState uint32

const (
	// DrainNotStarted means Core has not begun draining this service.
	DrainNotStarted DrainState = iota
	// DrainPending means native work remains; the caller's wait may be retried.
	DrainPending
	// DrainComplete means native work quiesced after admission closed. It does
	// not promise successful exchanges or peer application receipt of bytes.
	DrainComplete
	// DrainExpired means Core's native drain deadline forced unfinished work.
	// Cleanup may still be progressing; Stop joins it without changing the cause.
	DrainExpired
	// DrainAborted means an explicit stop prevented graceful completion.
	DrainAborted
	// DrainFailed means the owning native subsystem failed while draining.
	DrainFailed
	// DrainUnsupported means Core cannot certify this configured drain scope.
	DrainUnsupported
)

// DrainError keeps Core's observed state beside its typed operation error.
// A caller wait expiry (Pending) is distinct from Core expiry (Expired).
type DrainError struct {
	// State is the Core-issued observation, never a connector inference.
	State DrainState
	// Cause preserves the original error for errors.Is/errors.As.
	Cause error
}

// Error describes the unsuccessful drain observation without reclassifying it.
func (err *DrainError) Error() string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("CoAkka HTTP drain state %d: %v", err.State, err.Cause)
}

// Unwrap exposes Core's typed result to standard Go error inspection.
func (err *DrainError) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

// WaitDrained waits at most timeout (0..30 seconds) for Core's drain outcome.
// Zero inspects once. Call Drain first and keep readers/handlers progressing;
// this call never consumes events, stops Core, or releases borrowed results.
// Pending may be retried; terminal causes remain inspectable after Stop.
// Queued or borrowed immutable results do not prevent DrainComplete, but a
// borrowed lease still prevents destruction. Service.Close performs this
// protocol automatically; low-level Runtime.Close is forced resource cleanup.
func (runtime *Runtime) WaitDrained(timeout time.Duration) (DrainState, error) {
	if runtime == nil {
		return DrainNotStarted, errors.New("CoAkka HTTP Runtime is nil")
	}
	value, err := nativeDuration(timeout)
	if err != nil {
		return DrainNotStarted, err
	}
	runtime.lifecycle.Lock()
	defer runtime.lifecycle.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return DrainNotStarted, err
	}
	defer done.release()
	return native.waitDrained(value)
}
