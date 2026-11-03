package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import "errors"

func (native *nativeService) acceptWebSocket(exchange ExchangeID, subprotocol string) error {
	arena := nativeArena{}
	defer arena.close()
	value, err := arena.text(subprotocol)
	if err != nil {
		return err
	}
	return resultError(C.coakka_http_go_accept_websocket(native.api, native.service, nativeExchange(exchange), value))
}

func (native *nativeService) takeWebSocket(timeout C.uint64_t) (_ *SocketEvent, returned error) {
	var source C.coakka_http_host_websocket_event_t
	C.coakka_http_go_websocket_event_init(native.api, &source)
	if err := resultError(C.coakka_http_go_take_websocket(native.api, native.service, timeout, &source)); err != nil {
		return nil, err
	}
	defer func() {
		returned = errors.Join(returned, resultError(C.coakka_http_go_release_websocket(native.api, native.service, &source)))
	}()
	data, err := nativeBytes(source.data)
	if err != nil {
		return nil, err
	}
	return &SocketEvent{
		Kind:           SocketEventKind(source.kind),
		Session:        WebSocketID{Slot: uint64(source.session.slot), Generation: uint64(source.session.generation)},
		OriginExchange: ExchangeID{Slot: uint64(source.origin_exchange.slot), Generation: uint64(source.origin_exchange.generation)},
		CloseCode:      uint32(source.close_code),
		Data:           data,
	}, nil
}

func (native *nativeService) sendWebSocket(session WebSocketID, kind SocketEventKind, data []byte) error {
	arena := nativeArena{}
	defer arena.close()
	value, err := arena.bytes(data)
	if err != nil {
		return err
	}
	return resultError(C.coakka_http_go_send_websocket(
		native.api, native.service, nativeWebSocket(session),
		C.coakka_http_host_websocket_event_kind_t(kind), value))
}

func (native *nativeService) closeWebSocket(session WebSocketID, code uint32, reason string) error {
	arena := nativeArena{}
	defer arena.close()
	value, err := arena.text(reason)
	if err != nil {
		return err
	}
	return resultError(C.coakka_http_go_close_websocket(
		native.api, native.service, nativeWebSocket(session), C.uint32_t(code), value))
}

func (native *nativeService) submitOutbound(request ClientRequest) (OutboundID, error) {
	arena := nativeArena{}
	defer arena.close()
	var value C.coakka_http_host_outbound_request_t
	C.coakka_http_go_outbound_request_init(native.api, &value)
	value.timeout_ms = C.uint32_t(request.TimeoutMillis)
	var err error
	if value.logical_target, err = arena.text(request.LogicalTarget); err != nil {
		return OutboundID{}, err
	}
	if value.method, err = arena.text(request.Method); err != nil {
		return OutboundID{}, err
	}
	if value.target, err = arena.text(request.Target); err != nil {
		return OutboundID{}, err
	}
	if value.headers, value.header_count, err = arena.headers(request.Headers); err != nil {
		return OutboundID{}, err
	}
	if value.body, err = arena.bytes(request.Body); err != nil {
		return OutboundID{}, err
	}
	var call C.coakka_http_host_outbound_call_t
	if err = resultError(C.coakka_http_go_submit_outbound(native.api, native.service, &value, &call)); err != nil {
		return OutboundID{}, err
	}
	return OutboundID{Slot: uint64(call.slot), Generation: uint64(call.generation)}, nil
}

func (native *nativeService) cancelOutbound(call OutboundID) error {
	return resultError(C.coakka_http_go_cancel_outbound(native.api, native.service, nativeOutbound(call)))
}

func (native *nativeService) interruptOutbound() error {
	return resultError(C.coakka_http_go_interrupt_outbound(native.api, native.service))
}

func (native *nativeService) takeOutbound(timeout C.uint64_t) (_ *ClientTerminal, returned error) {
	var source C.coakka_http_host_outbound_event_t
	C.coakka_http_go_outbound_event_init(native.api, &source)
	status := C.coakka_http_go_take_outbound(native.api, native.service, timeout, &source)
	if status == C.COAKKA_HTTP_HOST_TIMEOUT {
		return nil, nil
	}
	if err := resultError(status); err != nil {
		return nil, err
	}
	defer func() {
		returned = errors.Join(returned, resultError(C.coakka_http_go_release_outbound(native.api, native.service, &source)))
	}()
	logicalTarget, err := nativeText(source.logical_target)
	if err != nil {
		return nil, err
	}
	selectedNode, err := nativeText(source.selected_node_id)
	if err != nil {
		return nil, err
	}
	body, err := nativeBytes(source.response_body)
	if err != nil {
		return nil, err
	}
	diagnostic, err := nativeText(source.diagnostic)
	if err != nil {
		return nil, err
	}
	count, err := checkedHeaderCount(source.response_header_count)
	if err != nil {
		return nil, err
	}
	entries := make([]Header, 0, count)
	for index := 0; index < count; index++ {
		var header C.coakka_http_host_header_t
		if C.coakka_http_go_outbound_header(native.api, native.service, &source, C.uint32_t(index), &header) == 0 {
			return nil, errors.New("native outbound header projection failed")
		}
		entry, copyErr := copiedHeader(header)
		if copyErr != nil {
			return nil, copyErr
		}
		entries = append(entries, entry)
	}
	headers := ownValidatedHeaders(entries)
	return &ClientTerminal{
		Reason:           OutboundReason(source.reason),
		Call:             OutboundID{Slot: uint64(source.call.slot), Generation: uint64(source.call.generation)},
		LogicalTarget:    logicalTarget,
		TargetGeneration: uint64(source.target_generation),
		SelectedNodeID:   selectedNode,
		Phase:            OutboundPhase(source.phase),
		Retry:            OutboundRetry(source.retry),
		Certainty:        OutboundCertainty(source.certainty),
		ResponseStatus:   uint16(source.response_status),
		ProviderCode:     int32(source.provider_code),
		ResponseHeaders:  headers,
		ResponseBody:     body,
		Diagnostic:       diagnostic,
	}, nil
}

func rebindCode(value C.coakka_http_host_control_code_t) RebindCode {
	return RebindCode(value)
}

// projectRequestRetirement copies Core truth on the cold control path. It
// never infers quiescence from local pending counts or terminal responses.
func projectRequestRetirement(value C.coakka_http_request_retirement_t) RequestRetirement {
	return RequestRetirement{
		State:            RequestRetirementState(value.state),
		Code:             RequestRetirementCode(value.code),
		QuiescentSources: uint32(value.quiescent_sources),
		Sources:          uint32(value.sources),
	}
}

func (native *nativeService) rebind(request RebindRequest) (RebindOutcome, error) {
	var value C.coakka_http_host_rebind_request_t
	C.coakka_http_go_rebind_init(native.api, &value)
	value.activation_id = C.uint64_t(request.ActivationID)
	value.expected_route_generation = C.uint64_t(request.ExpectedRouteGeneration)
	value.route_id = C.uint64_t(request.RouteID)
	value.expected_binding_revision = C.uint64_t(request.ExpectedBindingRevision)
	value.new_handler_binding_id = C.uint64_t(request.NewHandlerBindingID)
	var outcome C.coakka_http_host_rebind_outcome_t
	C.coakka_http_go_rebind_outcome_init(native.api, &outcome)
	if err := resultError(C.coakka_http_go_rebind(native.api, native.service, &value, &outcome)); err != nil {
		return RebindOutcome{}, err
	}
	return RebindOutcome{
		Code:                      rebindCode(outcome.code),
		RequestRetirement:         projectRequestRetirement(outcome.request_retirement),
		ActivationID:              uint64(outcome.activation_id),
		OperationDigest:           uint64(outcome.operation_digest),
		RouteGeneration:           uint64(outcome.route_generation),
		RouteID:                   uint64(outcome.route_id),
		PreviousHandlerBindingID:  uint64(outcome.previous_handler_binding_id),
		PreviousBindingRevision:   uint64(outcome.previous_binding_revision),
		EffectiveHandlerBindingID: uint64(outcome.effective_handler_binding_id),
		EffectiveBindingRevision:  uint64(outcome.effective_binding_revision),
		BindingChangeSequence:     uint64(outcome.binding_change_sequence),
		Changed:                   outcome.changed != 0,
		Replayed:                  outcome.replayed != 0,
	}, nil
}

func publicationCode(value C.coakka_http_host_control_code_t) RoutePublicationCode {
	return RoutePublicationCode(value)
}

func (native *nativeService) publishRoutes(request RoutePublication) (RoutePublicationOutcome, error) {
	if len(request.Routes) > maximumHandlerBindings {
		return RoutePublicationOutcome{}, errors.New("route generation exceeds the Go bound")
	}
	arena := nativeArena{}
	defer arena.close()
	routes, count, err := projectRoutes(native.api, &arena, request.Routes)
	if err != nil {
		return RoutePublicationOutcome{}, err
	}
	var value C.coakka_http_host_publication_request_t
	C.coakka_http_go_publication_init(native.api, &value)
	value.activation_id = C.uint64_t(request.ActivationID)
	value.expected_route_generation = C.uint64_t(request.ExpectedRouteGeneration)
	value.expected_metadata_generation = C.uint64_t(request.ExpectedMetadataGeneration)
	value.expected_binding_change_sequence = C.uint64_t(request.ExpectedBindingChangeSequence)
	value.routes = routes
	value.route_count = count
	var outcome C.coakka_http_host_publication_outcome_t
	C.coakka_http_go_publication_outcome_init(native.api, &outcome)
	if err = resultError(C.coakka_http_go_publish_routes(native.api, native.service, &value, &outcome)); err != nil {
		return RoutePublicationOutcome{}, err
	}
	return RoutePublicationOutcome{
		Code:                           publicationCode(outcome.code),
		RequestRetirement:              projectRequestRetirement(outcome.request_retirement),
		RouteCount:                     uint32(outcome.route_count),
		ActivationID:                   uint64(outcome.activation_id),
		OperationDigest:                uint64(outcome.operation_digest),
		PreviousRouteGeneration:        uint64(outcome.previous_route_generation),
		PreviousMetadataGeneration:     uint64(outcome.previous_metadata_generation),
		PreviousBindingChangeSequence:  uint64(outcome.previous_binding_change_sequence),
		EffectiveRouteGeneration:       uint64(outcome.effective_route_generation),
		EffectiveMetadataGeneration:    uint64(outcome.effective_metadata_generation),
		EffectiveBindingChangeSequence: uint64(outcome.effective_binding_change_sequence),
		RejectedRouteIndex:             uint32(outcome.rejected_route_index),
		Actual:                         uint64(outcome.actual),
		Limit:                          uint64(outcome.limit),
		Changed:                        outcome.changed != 0,
		Replayed:                       outcome.replayed != 0,
	}, nil
}

func copyHealth(source C.coakka_http_host_health_t) Health {
	return Health{
		RouteGeneration:                 uint64(source.route_generation),
		MetadataGeneration:              uint64(source.metadata_generation),
		Lifecycle:                       uint32(source.lifecycle),
		SnapshotSequence:                uint64(source.snapshot_sequence),
		ChangeSequence:                  uint64(source.binding_change_sequence),
		ObservedMonotonicNanos:          uint64(source.observed_monotonic_ns),
		ProgressSequence:                uint64(source.progress_sequence),
		ProgressMonotonicNanos:          uint64(source.progress_monotonic_ns),
		AcknowledgedProbeSequence:       uint64(source.acknowledged_probe_sequence),
		AcknowledgedProbeMonotonicNanos: uint64(source.acknowledged_probe_monotonic_ns),
		Ready:                           source.ready != 0,
		AdmissionOpen:                   source.admission_open != 0,
		Activity: ActivityInfo{
			ActiveRequests:    uint32(source.activity.active_requests),
			PendingRequests:   uint32(source.activity.pending_requests),
			ActiveConnections: uint32(source.activity.active_connections),
			WebSocketSessions: uint32(source.activity.websocket_sessions),
			OutboundPending:   uint32(source.activity.outbound_pending),
		},
	}
}

func (native *nativeService) health() (Health, error) {
	var value C.coakka_http_host_health_t
	C.coakka_http_go_health_init(native.api, &value)
	err := resultError(C.coakka_http_go_health(native.api, native.service, &value))
	return copyHealth(value), err
}

func (native *nativeService) probeLiveness(timeout C.uint64_t) (Health, error) {
	var value C.coakka_http_host_health_t
	C.coakka_http_go_health_init(native.api, &value)
	err := resultError(C.coakka_http_go_probe_liveness(native.api, native.service, timeout, &value))
	return copyHealth(value), err
}
