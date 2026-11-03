package coakkahttp

// Capabilities is the feature-bit set implemented by this connector.
type Capabilities uint64

const (
	// CapabilityInbound enables Runtime-owned inbound HTTP listeners.
	CapabilityInbound Capabilities = 1 << iota
	// CapabilityTLS enables server and client TLS.
	CapabilityTLS
	// CapabilityMutualTLS enables mutually authenticated TLS.
	CapabilityMutualTLS
	// CapabilityGZIP enables bounded gzip response compression.
	CapabilityGZIP
	// CapabilityRequestStream enables pulled request body chunks and trailers.
	CapabilityRequestStream
	// CapabilityResponseStream enables incremental response writes.
	CapabilityResponseStream
	// CapabilityResponseTrailers enables response-stream trailers.
	CapabilityResponseTrailers
	// CapabilityResponseWriteTimeout enables bounded response backpressure waits.
	CapabilityResponseWriteTimeout
	// CapabilityServerSentEvents enables typed SSE response writes.
	CapabilityServerSentEvents
	// CapabilityWebSocket enables Runtime-owned WebSocket sessions.
	CapabilityWebSocket
	// CapabilityHTTP2 enables HTTP/2 listeners or outbound policy.
	CapabilityHTTP2
	// CapabilityHTTP3 enables HTTP/3 listeners or outbound policy.
	CapabilityHTTP3
	// CapabilityStaticFiles enables Runtime-owned static file mounts.
	CapabilityStaticFiles
	// CapabilityApplicationFiles enables authority-scoped application file responses.
	CapabilityApplicationFiles
	// CapabilityOutbound enables the Runtime-owned HTTP client.
	CapabilityOutbound
	// CapabilityRouteRebind enables generation-checked handler rebinding.
	CapabilityRouteRebind
	// CapabilityOutboundLogicalTarget enables named outbound target topology.
	CapabilityOutboundLogicalTarget
	// CapabilityStaticFrontend enables index and SPA fallback behavior.
	CapabilityStaticFrontend
	// CapabilityHealth enables health snapshots and active liveness probes.
	CapabilityHealth
	// CapabilityMonitor enables bounded aggregate and event monitoring.
	CapabilityMonitor
	// CapabilityRouteManifestPublication enables complete structural route generations.
	CapabilityRouteManifestPublication
	// CapabilityAll is the complete capability set implemented by this connector.
	CapabilityAll = CapabilityRouteManifestPublication | (CapabilityRouteManifestPublication - 1)
)

// Has reports whether every requested feature bit is present.
func (capabilities Capabilities) Has(requested Capabilities) bool {
	return capabilities&requested == requested
}

// RebindCode preserves Core's public control code, including unknown values.
// Publication and request retirement have independent outcomes.
type RebindCode uint32

const (
	// RebindApplied means Core published or replayed the exact activation.
	RebindApplied RebindCode = 0
	// RebindInvalidArgument means one command field is invalid.
	RebindInvalidArgument RebindCode = 1
	// RebindClosed means the publication owner is closed.
	RebindClosed RebindCode = 2
	// RebindActivationConflict means the activation ID names a different command.
	RebindActivationConflict RebindCode = 3
	// RebindActivationExpired means the activation predates retained history; its result is not recoverable.
	RebindActivationExpired RebindCode = 4
	// RebindActivationOutOfOrder means activation ordering was violated.
	RebindActivationOutOfOrder RebindCode = 5
	// RebindRouteNotFound means the route ID does not exist.
	RebindRouteNotFound RebindCode = 6
	// RebindRouteGenerationMismatch means the structural generation is stale.
	RebindRouteGenerationMismatch RebindCode = 7
	// RebindBindingRevisionMismatch means the binding revision is stale.
	RebindBindingRevisionMismatch RebindCode = 8
	// RebindSequenceExhausted means a publication counter cannot advance.
	RebindSequenceExhausted RebindCode = 9
	// RebindRejected means Core rejected the candidate.
	RebindRejected RebindCode = 10
	// RebindOutOfMemory means Core could not own the candidate.
	RebindOutOfMemory RebindCode = 11
	// RebindInternal means Core rejected an internal invariant.
	RebindInternal RebindCode = 12
	// RebindTimeout means observation timed out; retry the exact activation and payload, never infer rejection.
	RebindTimeout RebindCode = 13
	// RebindWakeupFailed means observation failed; retry the exact activation and payload, never infer rejection.
	RebindWakeupFailed RebindCode = 14
	// RebindBusy means another activation is unresolved; this operation was not submitted.
	RebindBusy RebindCode = 15
)

// RoutePublicationCode preserves Core's public control code, including unknown values.
// Publication and request retirement have independent outcomes.
type RoutePublicationCode uint32

const (
	// RoutePublicationApplied means Core published or replayed the exact activation.
	RoutePublicationApplied RoutePublicationCode = 0
	// RoutePublicationInvalidArgument means one command field is invalid.
	RoutePublicationInvalidArgument RoutePublicationCode = 1
	// RoutePublicationClosed means the publication owner is closed.
	RoutePublicationClosed RoutePublicationCode = 2
	// RoutePublicationActivationConflict means the activation ID names a different command.
	RoutePublicationActivationConflict RoutePublicationCode = 3
	// RoutePublicationActivationExpired means the activation predates retained history; its result is not recoverable.
	RoutePublicationActivationExpired RoutePublicationCode = 4
	// RoutePublicationActivationOutOfOrder means activation ordering was violated.
	RoutePublicationActivationOutOfOrder RoutePublicationCode = 5
	// RoutePublicationRouteNotFound means the route ID does not exist.
	RoutePublicationRouteNotFound RoutePublicationCode = 6
	// RoutePublicationGenerationMismatch means the structural generation is stale.
	RoutePublicationGenerationMismatch RoutePublicationCode = 7
	// RoutePublicationRevisionMismatch means the binding revision is stale.
	RoutePublicationRevisionMismatch RoutePublicationCode = 8
	// RoutePublicationSequenceExhausted means a publication counter cannot advance.
	RoutePublicationSequenceExhausted RoutePublicationCode = 9
	// RoutePublicationRejected means Core rejected the candidate.
	RoutePublicationRejected RoutePublicationCode = 10
	// RoutePublicationOutOfMemory means Core could not own the candidate.
	RoutePublicationOutOfMemory RoutePublicationCode = 11
	// RoutePublicationInternal means Core rejected an internal invariant.
	RoutePublicationInternal RoutePublicationCode = 12
	// RoutePublicationTimeout means observation timed out; retry the exact activation and payload, never infer rejection.
	RoutePublicationTimeout RoutePublicationCode = 13
	// RoutePublicationWakeupFailed means observation failed; retry the exact activation and payload, never infer rejection.
	RoutePublicationWakeupFailed RoutePublicationCode = 14
	// RoutePublicationBusy means another activation is unresolved; this operation was not submitted.
	RoutePublicationBusy RoutePublicationCode = 15
)

// ListenerProtocol selects the inbound protocol owned by Runtime.
type ListenerProtocol uint32

const (
	// ProtocolHTTP11 selects HTTP/1.1.
	ProtocolHTTP11 ListenerProtocol = 1
	// ProtocolHTTP2 selects HTTP/2.
	ProtocolHTTP2 ListenerProtocol = 2
	// ProtocolHTTP3 selects HTTP/3.
	ProtocolHTTP3 ListenerProtocol = 3
)

// TransportSecurity selects plaintext, TLS, or mutual TLS.
type TransportSecurity uint32

const (
	// SecurityPlaintext disables transport encryption.
	SecurityPlaintext TransportSecurity = 1
	// SecurityTLS enables server-authenticated TLS.
	SecurityTLS TransportSecurity = 2
	// SecurityMutualTLS enables client and server authentication.
	SecurityMutualTLS TransportSecurity = 3
)

// BodyDelivery selects inline or pulled request-body delivery.
type BodyDelivery uint32

const (
	// BodyInline places the bounded request body on the request event.
	BodyInline BodyDelivery = 1
	// BodyStream emits request data, trailer, and end events.
	BodyStream BodyDelivery = 2
)

// CompressionMode selects response compression behavior.
type CompressionMode uint32

const (
	// CompressionDisabled disables response compression.
	CompressionDisabled CompressionMode = 1
	// CompressionGZIP enables bounded gzip compression.
	CompressionGZIP CompressionMode = 2
)

// IOBackend selects the native event I/O backend.
type IOBackend uint32

const (
	// IOPlatformDefault selects the platform's production backend.
	IOPlatformDefault IOBackend = 1
	// IOUring requests io_uring where supported.
	IOUring IOBackend = 2
)

// IOBackendFallbackReason explains why Runtime did not use a requested backend.
type IOBackendFallbackReason uint32

const (
	// IOFallbackNone means no backend fallback occurred.
	IOFallbackNone IOBackendFallbackReason = 0
	// IOFallbackConfiguration means the selected protocol rejected the backend.
	IOFallbackConfiguration IOBackendFallbackReason = 1
	// IOFallbackUnavailable means io_uring or the current host was unavailable.
	IOFallbackUnavailable IOBackendFallbackReason = 2
)

// OutboundStrategy selects endpoint ownership and balancing.
type OutboundStrategy uint32

const (
	// OutboundSingleOwner selects one endpoint owner.
	OutboundSingleOwner OutboundStrategy = 1
	// OutboundWeightedRoundRobin balances by endpoint weight.
	OutboundWeightedRoundRobin OutboundStrategy = 2
)

// MonitorCollection selects disabled, aggregate, or aggregate-plus-event collection.
type MonitorCollection uint32

const (
	// MonitorDisabled disables monitoring collection.
	MonitorDisabled MonitorCollection = 0
	// MonitorAggregates collects bounded counters and histograms.
	MonitorAggregates MonitorCollection = 1
	// MonitorAggregatesAndEvents also retains bounded operational events.
	MonitorAggregatesAndEvents MonitorCollection = 2
)

// EventKind identifies an event pulled from the inbound Runtime lane.
type EventKind uint32

const (
	// EventRequest begins an admitted exchange.
	EventRequest EventKind = 1
	// EventRequestData carries one streamed request chunk.
	EventRequestData EventKind = 2
	// EventRequestTrailers carries request trailers.
	EventRequestTrailers EventKind = 3
	// EventRequestEnd marks the end of a streamed request.
	EventRequestEnd EventKind = 4
	// EventRequestCancelled reports request cancellation.
	EventRequestCancelled EventKind = 5
	// EventResponseWritable reports released response-stream capacity.
	EventResponseWritable EventKind = 6
	// EventTerminal carries the final exchange summary.
	EventTerminal EventKind = 7
)

// SocketEventKind identifies a pulled or sent WebSocket event.
type SocketEventKind uint32

const (
	// WebSocketOpen reports an accepted session.
	WebSocketOpen SocketEventKind = 1
	// WebSocketText carries UTF-8 application data.
	WebSocketText SocketEventKind = 2
	// WebSocketBinary carries binary application data.
	WebSocketBinary SocketEventKind = 3
	// WebSocketPing carries a peer ping.
	WebSocketPing SocketEventKind = 4
	// WebSocketPong carries a peer pong.
	WebSocketPong SocketEventKind = 5
	// WebSocketWritable reports released outbound capacity.
	WebSocketWritable SocketEventKind = 6
	// WebSocketClose reports terminal session closure.
	WebSocketClose SocketEventKind = 7
	// WebSocketAcceptFailed reports failed protocol upgrade.
	WebSocketAcceptFailed SocketEventKind = 8
)

// MonitorNotification selects polling or a reserved signal notification.
type MonitorNotification uint32

const (
	// MonitorNotifyPoll requires polling or MonitorWait.
	MonitorNotifyPoll MonitorNotification = 0
	// MonitorNotifySignal requests the pre-reserved signal path.
	MonitorNotifySignal MonitorNotification = 1
)

// MonitorLatency selects latency collection.
type MonitorLatency uint32

const (
	// MonitorLatencyNone disables latency buckets.
	MonitorLatencyNone MonitorLatency = 0
	// MonitorLatencyFixed enables Runtime's fixed bounded buckets.
	MonitorLatencyFixed MonitorLatency = 1
)

// MonitorApplyReason explains an accepted or rejected live monitor policy.
type MonitorApplyReason uint32

const (
	// MonitorApplyApplied reports successful application.
	MonitorApplyApplied MonitorApplyReason = 0
	// MonitorApplyInvalidArgument reports malformed input.
	MonitorApplyInvalidArgument MonitorApplyReason = 1
	// MonitorApplyInvalidEnum reports an unknown enum value.
	MonitorApplyInvalidEnum MonitorApplyReason = 2
	// MonitorApplyUnsupportedCategory reports a category unavailable in this Runtime.
	MonitorApplyUnsupportedCategory MonitorApplyReason = 3
	// MonitorApplyInvalidCategoryRelationship reports inconsistent category sets.
	MonitorApplyInvalidCategoryRelationship MonitorApplyReason = 4
	// MonitorApplyResourceReservationExceeded reports a construction-time bound.
	MonitorApplyResourceReservationExceeded MonitorApplyReason = 5
	// MonitorApplySignalNotReserved reports an unreserved signal request.
	MonitorApplySignalNotReserved MonitorApplyReason = 6
	// MonitorApplyGenerationConflict reports optimistic-concurrency failure.
	MonitorApplyGenerationConflict MonitorApplyReason = 7
	// MonitorApplyInvalidLifecycle reports a disallowed lifecycle state.
	MonitorApplyInvalidLifecycle MonitorApplyReason = 8
	// MonitorApplySequenceExhausted reports sequence-space exhaustion.
	MonitorApplySequenceExhausted MonitorApplyReason = 9
	// MonitorApplyInternal reports an internal policy failure.
	MonitorApplyInternal MonitorApplyReason = 10
)

// MonitorEventKind identifies a retained operational monitor event.
type MonitorEventKind uint32

const (
	// MonitorEventLifecycleChanged reports a lifecycle transition.
	MonitorEventLifecycleChanged MonitorEventKind = 1
	// MonitorEventPolicyApplied reports a live policy update.
	MonitorEventPolicyApplied MonitorEventKind = 2
	// MonitorEventPolicyRejected reports a rejected live policy.
	MonitorEventPolicyRejected MonitorEventKind = 3
	// MonitorEventExchangeCompleted reports successful exchange completion.
	MonitorEventExchangeCompleted MonitorEventKind = 4
	// MonitorEventExchangeFailed reports exchange failure.
	MonitorEventExchangeFailed MonitorEventKind = 5
	// MonitorEventExchangeTimedOut reports exchange timeout.
	MonitorEventExchangeTimedOut MonitorEventKind = 6
	// MonitorEventExchangeDisconnected reports peer disconnection.
	MonitorEventExchangeDisconnected MonitorEventKind = 7
	// MonitorEventExchangeCancelled reports exchange cancellation.
	MonitorEventExchangeCancelled MonitorEventKind = 8
)

// MonitorCategory selects a bounded group of operational measurements.
type MonitorCategory uint32

const (
	// MonitorCategoryLifecycle selects lifecycle measurements.
	MonitorCategoryLifecycle MonitorCategory = 1 << iota
	// MonitorCategoryExchange selects exchange measurements.
	MonitorCategoryExchange
	// MonitorCategoryControl selects route and live-policy control measurements.
	MonitorCategoryControl
	// MonitorCategoryLatency selects latency measurements.
	MonitorCategoryLatency
)
