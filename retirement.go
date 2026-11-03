package coakkahttp

// RequestRetirementState preserves Core's request-capture state, including
// unknown numeric values. A zero value never permits handler disposal.
type RequestRetirementState uint32

const (
	// RetirementUnknown means no complete request-capture proof is available.
	RetirementUnknown RequestRetirementState = 0
	// RetirementCaptured means requests still retain the replaced identity.
	RetirementCaptured RequestRetirementState = 1
	// RetirementQuiescent certifies only Core request captures at this cut.
	RetirementQuiescent RequestRetirementState = 2
	// RetirementStillRoutable means the identity remains in the effective routes.
	RetirementStillRoutable RequestRetirementState = 3
	// RetirementNotApplicable means there is no replaced capture to retire.
	RetirementNotApplicable RequestRetirementState = 4
)

// RequestRetirementCode preserves the separate Core observation outcome,
// including unknown values. Publication success does not imply Observed.
type RequestRetirementCode uint32

const (
	// RetirementUnavailable means no observation was obtained.
	RetirementUnavailable RequestRetirementCode = 0
	// RetirementObserved means Core obtained a request-capture observation.
	RetirementObserved RequestRetirementCode = 1
	// RetirementInvalidArgument means the observation request was invalid.
	RetirementInvalidArgument RequestRetirementCode = 2
	// RetirementUnsupported means the selected owner cannot supply this proof.
	RetirementUnsupported RequestRetirementCode = 3
	// RetirementOutOfMemory means Core could not allocate observation resources.
	RetirementOutOfMemory RequestRetirementCode = 4
	// RetirementClosed means the observation owner is closed.
	RetirementClosed RequestRetirementCode = 5
	// RetirementBusy means an observation cannot currently proceed.
	RetirementBusy RequestRetirementCode = 6
	// RetirementConflict means the identity conflicts with retained intent.
	RetirementConflict RequestRetirementCode = 7
	// RetirementExpired means the required observation history expired.
	RetirementExpired RequestRetirementCode = 8
	// RetirementTimeout means the observation deadline expired, not publication rollback.
	RetirementTimeout RequestRetirementCode = 9
	// RetirementWakeupFailed means Core could not confirm the observation wakeup.
	RetirementWakeupFailed RequestRetirementCode = 10
	// RetirementPublicationFailed means publication supplied no retirement cut.
	RetirementPublicationFailed RequestRetirementCode = 11
	// RetirementObservationFailed means Core could not establish the observation.
	RetirementObservationFailed RequestRetirementCode = 12
	// RetirementNotFound means the requested capture identity was not found.
	RetirementNotFound RequestRetirementCode = 13
	// RetirementSystemError means an operating-system observation failed.
	RetirementSystemError RequestRetirementCode = 14
)

// RequestRetirement is a Core-issued proof tied to one publication. Exact
// control retry refreshes it without resubmitting intent. Even Quiescent does
// not release Go callbacks, sessions or application references. Republishing
// an old binding creates a new lifetime; this proof does not cover that use.
type RequestRetirement struct {
	// State is the request-capture state; Unknown is not proof of quiescence.
	State RequestRetirementState
	// Code is the observation outcome, independent of publication success.
	Code RequestRetirementCode
	// QuiescentSources counts owners acknowledging the captured identity.
	QuiescentSources uint32
	// Sources counts participating request owners, not CPUs or Go workers.
	Sources uint32
}
