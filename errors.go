package coakkahttp

import "fmt"

// ResultCode preserves the native status value, including values introduced by
// a newer compatible native image.
type ResultCode int32

// FailureReason is the stable cause of one failed host operation.
type FailureReason uint32

const (
	// ResultOK reports successful completion.
	ResultOK ResultCode = 0
	// ResultInvalidArgument reports a malformed caller value.
	ResultInvalidArgument ResultCode = 1
	// ResultInvalidMethod reports an invalid HTTP method.
	ResultInvalidMethod ResultCode = 2
	// ResultInvalidPattern reports an invalid route pattern.
	ResultInvalidPattern ResultCode = 3
	// ResultCapacityExhausted reports a configured resource bound was exhausted.
	ResultCapacityExhausted ResultCode = 4
	// ResultAmbiguous reports an ambiguous route declaration.
	ResultAmbiguous ResultCode = 5
	// ResultOutOfMemory reports allocation refusal.
	ResultOutOfMemory ResultCode = 6
	// ResultDuplicateRouteID reports a repeated route identity.
	ResultDuplicateRouteID ResultCode = 7
	// ResultDuplicateBindingID reports a repeated handler-binding identity.
	ResultDuplicateBindingID ResultCode = 8
	// ResultUnsupported reports a missing ABI or unavailable capability.
	ResultUnsupported ResultCode = 9
	// ResultClosed reports an already closed owner or lane.
	ResultClosed ResultCode = 10
	// ResultTimeout reports expiry of a monotonic timeout.
	ResultTimeout ResultCode = 11
	// ResultNotFound reports an unknown stable identifier.
	ResultNotFound ResultCode = 12
	// ResultIOError reports an operating-system I/O failure.
	ResultIOError ResultCode = 13
	// ResultStaleExchange reports that a captured exchange no longer admits
	// the operation. Core already owns its terminal outcome; never resubmit it.
	ResultStaleExchange ResultCode = 14
	// ResultControlUnresolved means Core cannot yet certify the current route
	// cut. Retry the pending activation; do not infer rollback or publish more.
	ResultControlUnresolved ResultCode = 15
)

const (
	// FailureReasonNone reports that no causal failure was supplied.
	FailureReasonNone FailureReason = iota
	// FailureReasonInvalidArgument reports invalid caller input.
	FailureReasonInvalidArgument
	// FailureReasonInvalidState reports an invalid lifecycle transition.
	FailureReasonInvalidState
	// FailureReasonCapacity reports a configured resource ceiling.
	FailureReasonCapacity
	// FailureReasonTimeout reports expiry of a monotonic deadline.
	FailureReasonTimeout
	// FailureReasonClosed reports an already closed owner or lane.
	FailureReasonClosed
	// FailureReasonOutOfMemory reports native allocation refusal.
	FailureReasonOutOfMemory
	// FailureReasonUnsupported reports an unavailable capability.
	FailureReasonUnsupported
	// FailureReasonListenerStart reports failure to start an HTTP listener.
	FailureReasonListenerStart
	// FailureReasonTransportSecurity reports listener security setup failure.
	FailureReasonTransportSecurity
	// FailureReasonIOBackend reports I/O backend setup failure.
	FailureReasonIOBackend
	// FailureReasonInternal reports an invariant failure inside Runtime.
	FailureReasonInternal
)

// Error is a copied Runtime failure. Actual and Limit describe a rejected bound
// when Runtime supplied that information.
type Error struct {
	// Code is the stable machine-readable result.
	Code ResultCode
	// Reason preserves the machine-readable cause, including newer values.
	Reason FailureReason
	// SystemCode is the immediate operating-system error, or zero when absent.
	SystemCode int32
	// Actual is the observed size or count for a bound failure.
	Actual uint64
	// Limit is the configured ceiling for a bound failure.
	Limit uint64
	// Detail is a bounded diagnostic safe after the native call returns.
	Detail string
}

// Error returns the bounded Runtime diagnostic and stable result code.
func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Detail != "" {
		if err.Actual != 0 || err.Limit != 0 {
			return fmt.Sprintf("%s (actual=%d limit=%d)", err.Detail, err.Actual, err.Limit)
		}
		return err.Detail
	}
	return fmt.Sprintf("CoAkka HTTP operation failed with code %d", err.Code)
}

// Is reports whether target is a Runtime error with the same result code.
func (err *Error) Is(target error) bool {
	other, ok := target.(*Error)
	return ok && err != nil && err.Code == other.Code
}
