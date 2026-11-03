package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import "fmt"

// OutboundReason is Core's causal terminal result, not an HTTP status.
// Unknown values remain representable and must not be treated as success.
type OutboundReason uint32

const (
	// OutboundResponse means a complete HTTP response, including a 4xx/5xx.
	OutboundResponse OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_RESPONSE
	// OutboundProviderFailure means the transport failed after adoption.
	OutboundProviderFailure OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_PROVIDER_FAILURE
	// OutboundProviderRejected means the transport refused adoption.
	OutboundProviderRejected OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_PROVIDER_REJECTED
	// OutboundCancelled means Core terminalized an explicit cancellation.
	OutboundCancelled OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_CANCELLED
	// OutboundDeadlineExceeded means Core's monotonic call budget expired.
	OutboundDeadlineExceeded OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_DEADLINE_EXCEEDED
	// OutboundStopped means shutdown terminalized the call.
	OutboundStopped OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_STOPPED
	// OutboundResponseHeadLimit means the response head exceeded its bound.
	OutboundResponseHeadLimit OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_RESPONSE_HEAD_LIMIT
	// OutboundResponseBodyLimit means the response body exceeded its bound.
	OutboundResponseBodyLimit OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_RESPONSE_BODY_LIMIT
	// OutboundResponseAggregateLimit means aggregate retention was exhausted.
	OutboundResponseAggregateLimit OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_RESPONSE_AGGREGATE_LIMIT
	// OutboundOutOfMemory means Core could not retain the result.
	OutboundOutOfMemory OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_OUT_OF_MEMORY
	// OutboundClockFailure means the deadline clock failed.
	OutboundClockFailure OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_CLOCK_FAILURE
	// OutboundInvalidProviderResult means the provider violated its contract.
	OutboundInvalidProviderResult OutboundReason = C.COAKKA_HTTP_HOST_OUTBOUND_INVALID_PROVIDER_RESULT
)

// String derives a stable symbolic code without storing a duplicate reason name.
func (reason OutboundReason) String() string {
	switch reason {
	case OutboundResponse:
		return "RESPONSE"
	case OutboundProviderFailure:
		return "PROVIDER_FAILURE"
	case OutboundProviderRejected:
		return "PROVIDER_REJECTED"
	case OutboundCancelled:
		return "CANCELLED"
	case OutboundDeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case OutboundStopped:
		return "STOPPED"
	case OutboundResponseHeadLimit:
		return "RESPONSE_HEAD_LIMIT"
	case OutboundResponseBodyLimit:
		return "RESPONSE_BODY_LIMIT"
	case OutboundResponseAggregateLimit:
		return "RESPONSE_AGGREGATE_LIMIT"
	case OutboundOutOfMemory:
		return "OUT_OF_MEMORY"
	case OutboundClockFailure:
		return "CLOCK_FAILURE"
	case OutboundInvalidProviderResult:
		return "INVALID_PROVIDER_RESULT"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", uint32(reason))
	}
}

// OutboundPhase identifies Core's causal phase; unknown values are preserved.
type OutboundPhase uint32

const (
	// OutboundPhaseAdmission identifies initial admission.
	OutboundPhaseAdmission OutboundPhase = C.COAKKA_HTTP_HOST_OUTBOUND_PHASE_ADMISSION
	// OutboundPhaseProviderAdoption identifies transport adoption.
	OutboundPhaseProviderAdoption OutboundPhase = C.COAKKA_HTTP_HOST_OUTBOUND_PHASE_PROVIDER_ADOPTION
	// OutboundPhaseTransport identifies in-flight transport work.
	OutboundPhaseTransport OutboundPhase = C.COAKKA_HTTP_HOST_OUTBOUND_PHASE_TRANSPORT
	// OutboundPhaseResponse identifies response processing.
	OutboundPhaseResponse OutboundPhase = C.COAKKA_HTTP_HOST_OUTBOUND_PHASE_RESPONSE
	// OutboundPhaseCancellation identifies cancellation processing.
	OutboundPhaseCancellation OutboundPhase = C.COAKKA_HTTP_HOST_OUTBOUND_PHASE_CANCELLATION
	// OutboundPhaseShutdown identifies shutdown processing.
	OutboundPhaseShutdown OutboundPhase = C.COAKKA_HTTP_HOST_OUTBOUND_PHASE_SHUTDOWN
)

// OutboundRetry is Core's disposition, not permission to replay an operation.
// The application still owns idempotency, authorization and its retry budget.
type OutboundRetry uint32

const (
	// OutboundRetryNever prohibits retry for this result.
	OutboundRetryNever OutboundRetry = C.COAKKA_HTTP_HOST_OUTBOUND_RETRY_NEVER
	// OutboundRetryIfSafe requires application-level replay safety.
	OutboundRetryIfSafe OutboundRetry = C.COAKKA_HTTP_HOST_OUTBOUND_RETRY_IF_SAFE
	// OutboundRetryDeliveryUncertain warns that the peer may have acted.
	OutboundRetryDeliveryUncertain OutboundRetry = C.COAKKA_HTTP_HOST_OUTBOUND_RETRY_DELIVERY_UNCERTAIN
)

// OutboundCertainty is Core-observed delivery evidence, not connector inference.
type OutboundCertainty uint32

const (
	// OutboundNoRequestBytesSent means no request bytes were sent.
	OutboundNoRequestBytesSent OutboundCertainty = C.COAKKA_HTTP_HOST_OUTBOUND_NO_REQUEST_BYTES_SENT
	// OutboundRequestUncertain means delivery cannot be established.
	OutboundRequestUncertain OutboundCertainty = C.COAKKA_HTTP_HOST_OUTBOUND_REQUEST_UNCERTAIN
	// OutboundResponseHeadObserved means a response head was observed.
	OutboundResponseHeadObserved OutboundCertainty = C.COAKKA_HTTP_HOST_OUTBOUND_RESPONSE_HEAD_OBSERVED
	// OutboundCompleteResponse means the complete response was observed.
	OutboundCompleteResponse OutboundCertainty = C.COAKKA_HTTP_HOST_OUTBOUND_COMPLETE_RESPONSE
)
