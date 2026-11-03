package coakkahttp

// ExecutionMode identifies Core's accepted construction mode. It is observation
// only; applications cannot override the connector's scheduling profile.
// Unknown values are preserved for diagnostics, never mapped to a default.
type ExecutionMode uint32

const (
	// ExecutionDefault means Core received no explicit execution preference.
	ExecutionDefault ExecutionMode = 0
	// ExecutionExact means Core accepted an exact topology request.
	ExecutionExact ExecutionMode = 1
	// ExecutionPreferSupported means Core admitted a preferred topology.
	ExecutionPreferSupported ExecutionMode = 2
)

// ExecutionReason is Core's construction decision, not a connector diagnosis.
// Unknown numeric values are retained without reclassification.
type ExecutionReason uint32

const (
	// ExecutionDefaulted identifies Core's default topology.
	ExecutionDefaulted ExecutionReason = 0
	// ExecutionExactSelected identifies an accepted exact topology.
	ExecutionExactSelected ExecutionReason = 1
	// ExecutionPreferredSelected identifies the accepted preferred topology.
	ExecutionPreferredSelected ExecutionReason = 2
	// ExecutionFacilityRestricted identifies a configured facility restriction.
	ExecutionFacilityRestricted ExecutionReason = 3
	// ExecutionBudgetRestricted identifies a resource-budget restriction.
	ExecutionBudgetRestricted ExecutionReason = 4
	// ExecutionCPURestricted identifies a Core-observed CPU allowance restriction.
	ExecutionCPURestricted ExecutionReason = 5
	// ExecutionCPUUnobserved means Core could not establish a CPU allowance.
	ExecutionCPUUnobserved ExecutionReason = 6
)

// ExecutionSelection is Core's immutable construction decision. Interpret it
// only when the enclosing ExecutionInfo.Observed is true. Counts are HTTP loops,
// not CPUs or Go workers; they do not change during start, drain or stop.
type ExecutionSelection struct {
	// RequestedMode is the construction mode accepted by Core.
	RequestedMode ExecutionMode
	// RequestedCount is the preferred count; zero only for ExecutionDefault.
	RequestedCount uint32
	// SelectedCount is Core's accepted reservation, not current activity.
	SelectedCount uint32
	// Reason explains Core's selection without inferring from local settings.
	Reason ExecutionReason
}

// ExecutionInfo is Core's cold, bounded observation of HTTP loop state. It is
// not a simultaneous cross-thread barrier, CPU utilization, or a lease-release
// certificate. No instance inspected means Observed=false, not a live zero.
type ExecutionInfo struct {
	// Selection reports Core's immutable construction decision.
	Selection ExecutionSelection
	// ConfiguredEventLoops is the accepted reservation, stable across stop/failure.
	ConfiguredEventLoops uint32
	// ActiveEventLoops counts Core loops observed running or draining.
	ActiveEventLoops uint32
	// DrainingEventLoops is the draining subset of ActiveEventLoops.
	DrainingEventLoops uint32
	// FailedEventLoops counts currently failed loops, not historical failures.
	FailedEventLoops uint32
	// Observed distinguishes a service observation from an uninspected instance.
	Observed bool
}
