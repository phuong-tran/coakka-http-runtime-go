package coakkahttp

// CloseError preserves the close cause and whether native ownership remains.
// Use errors.As to inspect it; errors.As also reaches the underlying *Error.
// RuntimeRetained=false means cleanup finished despite an error (for example,
// CPU restoration failed after destruction); retry cannot restore that owner.
type CloseError struct {
	// Err is the original bounded shutdown/cleanup failure, not a replacement code.
	Err error
	// RuntimeRetained means the same owner must remain reachable for a Close retry.
	RuntimeRetained bool
}

// Error returns the original cleanup diagnostic.
func (failure *CloseError) Error() string { return failure.Err.Error() }

// Unwrap retains Go's errors.Is/errors.As behavior for the native cause.
func (failure *CloseError) Unwrap() error { return failure.Err }

func closeFailure(err error, retained bool) error {
	if err == nil {
		return nil
	}
	return &CloseError{Err: err, RuntimeRetained: retained}
}

// StartError reports a failed Builder startup whose cleanup also failed.
// Normal startup failures with successful cleanup retain their original error.
type StartError struct {
	// Err preserves both startup and cleanup causes for errors.Is/errors.As.
	Err error
	// RetainedRuntime is non-nil only if Core refused destruction. No handlers
	// or Service workers have started. Retry RetainedRuntime.Close; do not start
	// it or abandon it. A post-destruction cleanup error leaves this field nil.
	RetainedRuntime *Runtime
}

// Error returns the joined startup and cleanup diagnostics.
func (failure *StartError) Error() string { return failure.Err.Error() }

// Unwrap preserves access to the native startup and cleanup causes.
func (failure *StartError) Unwrap() error { return failure.Err }
