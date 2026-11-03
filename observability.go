package coakkahttp

// RebindRequest performs generation-checked handler rebinding.
type RebindRequest struct {
	// ActivationID is the idempotency identifier.
	ActivationID uint64
	// ExpectedRouteGeneration guards the route table generation.
	ExpectedRouteGeneration uint64
	// RouteID selects the route.
	RouteID uint64
	// ExpectedBindingRevision guards the current binding.
	ExpectedBindingRevision uint64
	// NewHandlerBindingID selects the replacement application binding.
	NewHandlerBindingID uint64
}

// RebindOutcome is the copied result of one route rebind command.
type RebindOutcome struct {
	// Code is Runtime's typed rebind outcome.
	Code RebindCode
	// ActivationID echoes the idempotency identifier.
	ActivationID uint64
	// OperationDigest identifies the normalized operation.
	OperationDigest uint64
	// RouteGeneration is the effective route generation.
	RouteGeneration uint64
	// RouteID is the affected route.
	RouteID uint64
	// PreviousHandlerBindingID is the replaced binding.
	PreviousHandlerBindingID uint64
	// PreviousBindingRevision is the prior revision.
	PreviousBindingRevision uint64
	// EffectiveHandlerBindingID is the resulting binding.
	EffectiveHandlerBindingID uint64
	// EffectiveBindingRevision is the resulting revision.
	EffectiveBindingRevision uint64
	// BindingChangeSequence is the resulting change sequence.
	BindingChangeSequence uint64
	// RequestRetirement is Core's request-capture proof, not handler/WS lifetime.
	// Retry the exact activation to refresh it without resubmitting publication.
	RequestRetirement RequestRetirement
	// Changed reports whether the binding changed.
	Changed bool
	// Replayed reports an idempotent history replay.
	Replayed bool
}

// RoutePublication replaces the complete structural route generation.
// Routes and their strings are borrowed only for PublishRoutes.
type RoutePublication struct {
	// ActivationID is the single-assignment idempotency identity.
	ActivationID uint64
	// ExpectedRouteGeneration guards structural route truth.
	ExpectedRouteGeneration uint64
	// ExpectedMetadataGeneration guards route-documentation truth.
	ExpectedMetadataGeneration uint64
	// ExpectedBindingChangeSequence guards handler binding truth.
	ExpectedBindingChangeSequence uint64
	// Routes is the complete replacement generation, not a delta.
	Routes []Route
}

// RoutePublicationOutcome is the copied result of one structural publication.
type RoutePublicationOutcome struct {
	// Code is Runtime's typed publication outcome.
	Code RoutePublicationCode
	// ActivationID echoes the operation identity.
	ActivationID uint64
	// OperationDigest identifies the normalized complete candidate.
	OperationDigest uint64
	// PreviousRouteGeneration is the structural generation before the call.
	PreviousRouteGeneration uint64
	// PreviousMetadataGeneration is the documentation generation before the call.
	PreviousMetadataGeneration uint64
	// PreviousBindingChangeSequence is the binding sequence before the call.
	PreviousBindingChangeSequence uint64
	// EffectiveRouteGeneration is the resulting structural generation.
	EffectiveRouteGeneration uint64
	// EffectiveMetadataGeneration is the resulting documentation generation.
	EffectiveMetadataGeneration uint64
	// EffectiveBindingChangeSequence is the resulting binding sequence.
	EffectiveBindingChangeSequence uint64
	// RequestRetirement is Core's proof for replaced structural captures. It
	// never certifies Go callback or WebSocket lifetime; exact retry refreshes it.
	RequestRetirement RequestRetirement
	// RouteCount is Core's effective count at this publication cut, not a
	// reconstruction from the input. It is unknown (zero) on ambiguous outcomes.
	RouteCount uint32
	// Actual is Core's observed quantity for a bounded refusal.
	Actual uint64
	// Limit is Core's authoritative ceiling for a bounded refusal.
	Limit uint64
	// RejectedRouteIndex identifies a rejected candidate route when applicable.
	RejectedRouteIndex uint32
	// Changed reports whether a new generation became effective.
	Changed bool
	// Replayed reports an exact idempotency-history replay.
	Replayed bool
}

// ActivityInfo is a cold, copied set of Core-owned gauges, not Go counters.
type ActivityInfo struct {
	// ActiveRequests counts retained active Core HTTP requests.
	ActiveRequests uint32
	// PendingRequests counts Core requests awaiting delivery.
	PendingRequests uint32
	// ActiveConnections counts Core's active HTTP connections.
	ActiveConnections uint32
	// WebSocketSessions counts Core-owned WebSocket sessions.
	WebSocketSessions uint32
	// OutboundPending counts Core's pending outbound calls.
	OutboundPending uint32
}

// Health contains only facts supplied by Core's public health contract.
// The snapshot is cold and must be discarded when its query returns an error.
type Health struct {
	// RouteGeneration is the coherent structural cut with ChangeSequence.
	RouteGeneration uint64
	// MetadataGeneration is Core's independent route-documentation generation.
	MetadataGeneration uint64
	// Lifecycle is Core's current lifecycle value, including unknown values.
	Lifecycle uint32
	// SnapshotSequence is the Core snapshot version.
	SnapshotSequence uint64
	// ChangeSequence is Core's binding-table change sequence.
	ChangeSequence uint64
	// ObservedMonotonicNanos is the Core snapshot timestamp.
	ObservedMonotonicNanos uint64
	// ProgressSequence is the latest Core forward-progress counter.
	ProgressSequence uint64
	// ProgressMonotonicNanos is the latest Core progress timestamp.
	ProgressMonotonicNanos uint64
	// AcknowledgedProbeSequence is the last Core liveness acknowledgement.
	AcknowledgedProbeSequence uint64
	// AcknowledgedProbeMonotonicNanos is its monotonic timestamp.
	AcknowledgedProbeMonotonicNanos uint64
	// Ready is Core's readiness observation, not a derived Go health verdict.
	Ready bool
	// AdmissionOpen reports Core's inbound admission state.
	AdmissionOpen bool
	// Activity groups Core's cold gauges; it is not a retirement certificate.
	Activity ActivityInfo
}

// MonitorPolicy is the live mutable subset of monitoring configuration.
type MonitorPolicy struct {
	// Collection selects disabled, aggregate, or event collection.
	Collection MonitorCollection
	// Notification selects polling or a reserved signal path.
	Notification MonitorNotification
	// Latency selects fixed histogram collection.
	Latency MonitorLatency
	// ActiveEventCapacity selects active slots within the reservation.
	ActiveEventCapacity uint32
	// AggregateCategories selects aggregate categories.
	AggregateCategories MonitorCategory
	// EventCategories selects retained event categories.
	EventCategories MonitorCategory
}

// MonitorConfig is the copied effective monitoring configuration.
type MonitorConfig struct {
	// Generation is used for optimistic live updates.
	Generation uint64
	// CollectionEpoch identifies the current collection epoch.
	CollectionEpoch uint64
	// ChangeSequence is the latest observed runtime change.
	ChangeSequence uint64
	// EpochStartedMonotonicNanos is the epoch start timestamp.
	EpochStartedMonotonicNanos uint64
	// PolicyAppliedMonotonicNanos is the policy application timestamp.
	PolicyAppliedMonotonicNanos uint64
	// SupportedCategories is Runtime's supported category bit set.
	SupportedCategories MonitorCategory
	// ReservedEventCapacity is the construction-time slot reservation.
	ReservedEventCapacity uint32
	// MaxEventsPerRead is the construction-time page bound.
	MaxEventsPerRead uint32
	// FixedLatencyBuckets reports whether histogram storage was reserved.
	FixedLatencyBuckets bool
	// SignalReserved reports whether signaling was reserved.
	SignalReserved bool
	// Policy is the effective live policy.
	Policy MonitorPolicy
}

// MonitorApplyOutcome is a typed accepted or rejected live policy result.
type MonitorApplyOutcome struct {
	// Reason explains application or rejection.
	Reason MonitorApplyReason
	// Actual is the observed value for a bound rejection.
	Actual uint64
	// Limit is the reserved bound for a rejection.
	Limit uint64
	// Changed reports whether the policy changed.
	Changed bool
	// Effective is the effective configuration after the attempt.
	Effective MonitorConfig
}

// MonitorFailureCount is one bounded failure counter.
type MonitorFailureCount struct {
	// Reason is Runtime's failure reason.
	Reason uint32
	// Count is the cumulative count for the current epoch.
	Count uint64
}

// MonitorLatencyBucket is one fixed histogram bucket.
type MonitorLatencyBucket struct {
	// UpperBoundNanos is the inclusive bucket ceiling.
	UpperBoundNanos uint64
	// Count is the cumulative bucket count.
	Count uint64
}

// MonitorSnapshot is a copied aggregate and health observation.
type MonitorSnapshot struct {
	// Collecting reports active collection.
	Collecting bool
	// Stale reports a snapshot behind current truth.
	Stale bool
	// Partial reports incomplete values.
	Partial bool
	// SnapshotSequence identifies the snapshot.
	SnapshotSequence uint64
	// ObservedMonotonicNanos is the observation timestamp.
	ObservedMonotonicNanos uint64
	// CollectedChangeSequence is the latest included change.
	CollectedChangeSequence uint64
	// InboundAdmittedRequests counts admitted requests.
	InboundAdmittedRequests uint64
	// InboundRequestBytes counts admitted request bytes.
	InboundRequestBytes uint64
	// ExchangeCompleted counts completed exchanges.
	ExchangeCompleted uint64
	// ExchangeFailed counts failed exchanges.
	ExchangeFailed uint64
	// ExchangeTimedOut counts timed-out exchanges.
	ExchangeTimedOut uint64
	// ExchangeDisconnected counts disconnected exchanges.
	ExchangeDisconnected uint64
	// ExchangeCancelled counts cancelled exchanges.
	ExchangeCancelled uint64
	// ResponseStatusFamily contains 1xx through 5xx counts.
	ResponseStatusFamily [5]uint64
	// RetainedEvents is the current event ring occupancy.
	RetainedEvents uint64
	// EventOverwriteCount counts overwritten retained events.
	EventOverwriteCount uint64
	// EventDropCount counts dropped events.
	EventDropCount uint64
	// SignalNotifyCount counts notification attempts.
	SignalNotifyCount uint64
	// SignalCoalescedCount counts coalesced notifications.
	SignalCoalescedCount uint64
	// SignalFailureCount counts notification failures.
	SignalFailureCount uint64
	// Config is the effective monitoring configuration.
	Config MonitorConfig
	// Health is the co-observed health snapshot.
	Health Health
	// FailureCounts contains the populated bounded failure counters.
	FailureCounts []MonitorFailureCount
	// LatencyBuckets contains the populated fixed histogram buckets.
	LatencyBuckets []MonitorLatencyBucket
}

// MonitorEvent is one copied pointer-free operational event.
type MonitorEvent struct {
	// Kind identifies the event.
	Kind MonitorEventKind
	// Sequence is the event cursor.
	Sequence uint64
	// ChangeSequence is the associated runtime change.
	ChangeSequence uint64
	// ConfigGeneration is the monitoring configuration generation.
	ConfigGeneration uint64
	// CollectionEpoch is the event collection epoch.
	CollectionEpoch uint64
	// ObservedMonotonicNanos is the event timestamp.
	ObservedMonotonicNanos uint64
	// Category is the event category bit.
	Category uint64
	// FailureReason is an optional Runtime failure reason.
	FailureReason uint32
	// QueueScope identifies an optional bounded queue.
	QueueScope uint32
	// QueueDepth is the observed queue occupancy.
	QueueDepth uint32
	// QueueCapacity is the configured queue ceiling.
	QueueCapacity uint32
	// Exchange is the optional associated exchange.
	Exchange ExchangeID
	// Detail is bounded runtime-safe detail, never application payload.
	Detail string
}

// MonitorEventPage is one cursor-based bounded monitoring page.
type MonitorEventPage struct {
	// Events contains at most the requested and reserved maximum.
	Events []MonitorEvent
	// OldestAvailableSequence is the oldest retained cursor.
	OldestAvailableSequence uint64
	// LatestSequence is the latest retained cursor.
	LatestSequence uint64
	// MissedEvents counts overwritten events after the supplied cursor.
	MissedEvents uint64
	// RemainingEvents counts newer retained events not in this page.
	RemainingEvents uint64
}
