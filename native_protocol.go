package coakkahttp

/*
#include "bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"runtime"
	"unsafe"
)

func nativeExchange(value ExchangeID) C.coakka_http_host_exchange_t {
	return C.coakka_http_host_exchange_t{slot: C.uint64_t(value.Slot), generation: C.uint64_t(value.Generation)}
}

func nativeWebSocket(value WebSocketID) C.coakka_http_host_websocket_t {
	return C.coakka_http_host_websocket_t{slot: C.uint64_t(value.Slot), generation: C.uint64_t(value.Generation)}
}

func nativeOutbound(value OutboundID) C.coakka_http_host_outbound_call_t {
	return C.coakka_http_host_outbound_call_t{slot: C.uint64_t(value.Slot), generation: C.uint64_t(value.Generation)}
}

func (arena *nativeArena) headers(headers Headers) (*C.coakka_http_host_header_t, C.uint32_t, error) {
	entries := headers.entries
	if len(entries) == 0 {
		return nil, 0, nil
	}
	descriptorBytes, total, err := headerProjectionSize(entries)
	if err != nil {
		return nil, 0, err
	}
	// One C-owned region holds the aligned descriptors followed by their bytes.
	// No Go pointer is stored in C. The arena frees the entire region after the
	// synchronous Core call has accepted/copied or refused it, on every path.
	memory, err := arena.allocate(uintptr(total))
	if err != nil {
		return nil, 0, err
	}
	values := unsafe.Slice((*C.coakka_http_host_header_t)(memory), len(entries))
	offset := descriptorBytes
	for index, entry := range entries {
		values[index].name = copyHeaderText(memory, &offset, entry.Name)
		values[index].value = copyHeaderText(memory, &offset, entry.Value)
	}
	return (*C.coakka_http_host_header_t)(memory), C.uint32_t(len(entries)), nil
}

// headerProjectionSize checks every range before allocation or pointer math.
// Per-field/count bounds are unchanged; the additional sum check prevents int
// overflow on narrower hosts. This is bounded O(n) enumeration, not lookup or
// duplicate protocol validation. No native allocation occurs on refusal.
func headerProjectionSize(entries []Header) (descriptors, total int, err error) {
	if len(entries) > maximumHeaders {
		return 0, 0, errors.New("HTTP header count exceeds the Go bound")
	}
	descriptors = len(entries) * int(unsafe.Sizeof(C.coakka_http_host_header_t{}))
	total = descriptors
	const maxInt = int(^uint(0) >> 1)
	for _, entry := range entries {
		for _, size := range [...]int{len(entry.Name), len(entry.Value)} {
			if size > maximumBridgeBytes || size > maxInt-total {
				return 0, 0, errors.New("native header projection exceeds the Go byte bound")
			}
			total += size
		}
	}
	return descriptors, total, nil
}

// The complete layout was checked before entry; offset advances only within
// that one C allocation. Empty text retains a nil view. copy handles arbitrary
// octets without a temporary Go byte slice or a separate cgo allocation call.
func copyHeaderText(memory unsafe.Pointer, offset *int, text string) C.coakka_http_host_bytes_t {
	if len(text) == 0 {
		return C.coakka_http_host_bytes_t{}
	}
	pointer := unsafe.Add(memory, *offset)
	copy(unsafe.Slice((*byte)(pointer), len(text)), text)
	*offset += len(text)
	return C.coakka_http_host_bytes_t{data: (*C.uint8_t)(pointer), size: C.size_t(len(text))}
}

func nativeResponse(api *C.coakka_http_go_api_t, arena *nativeArena, source Response) (C.coakka_http_host_response_t, error) {
	var value C.coakka_http_host_response_t
	C.coakka_http_go_response_init(api, &value)
	if source.Status < 100 || source.Status > 599 {
		return value, errors.New("HTTP response status is invalid")
	}
	value.status_code = C.uint32_t(source.Status)
	headers, count, err := arena.headers(source.Headers)
	if err != nil {
		return value, err
	}
	value.headers = headers
	value.header_count = count
	if value.body, err = arena.bytes(source.body); err != nil {
		return value, err
	}
	return value, nil
}

func (native *nativeService) start() error {
	var issue C.coakka_http_host_issue_t
	C.coakka_http_go_issue_init(native.api, &issue)
	return issueError(
		C.coakka_http_go_service_start(native.api, native.service, &issue),
		&issue,
	)
}

func (native *nativeService) boundPort() (uint16, error) {
	var port C.uint16_t
	err := resultError(C.coakka_http_go_service_port(native.api, native.service, &port))
	return uint16(port), err
}

func (native *nativeService) drain() error {
	return resultError(C.coakka_http_go_service_drain(native.api, native.service))
}

func (native *nativeService) waitDrained(timeout C.uint64_t) (DrainState, error) {
	var state C.coakka_http_host_drain_state_t
	err := resultError(C.coakka_http_go_service_wait_drained(native.api, native.service, timeout, &state))
	if err != nil {
		return DrainState(state), &DrainError{State: DrainState(state), Cause: err}
	}
	return DrainState(state), nil
}

func (native *nativeService) stop() error {
	return resultError(C.coakka_http_go_service_stop(native.api, native.service))
}

func (native *nativeService) interrupt() error {
	return resultError(C.coakka_http_go_interrupt_requests(native.api, native.service))
}

func (native *nativeService) takeEvent(timeout C.uint64_t) (_ *Event, returned error) {
	var source C.coakka_http_host_request_event_t
	// The private take bridge initializes through Core before waiting. It
	// transfers a lease only on success; every successful projection releases it.
	if err := resultError(C.coakka_http_go_take_request(native.api, native.service, timeout, &source)); err != nil {
		return nil, err
	}
	defer func() {
		returned = errors.Join(returned, resultError(C.coakka_http_go_release_request(native.api, native.service, &source)))
	}()
	data, err := nativeBytes(source.data)
	if err != nil {
		return nil, err
	}
	event := &Event{
		Kind:             EventKind(source.kind),
		Exchange:         ExchangeID{Slot: uint64(source.exchange.slot), Generation: uint64(source.exchange.generation)},
		Sequence:         uint64(source.sequence),
		HandlerBindingID: uint64(source.handler_binding_id),
		Data:             data,
	}
	switch event.Kind {
	case EventRequest:
		event.Request, err = native.copyRequest(&source, event.Exchange, event.HandlerBindingID)
	case EventRequestTrailers:
		event.StreamTrailers, err = native.copyStreamTrailers(&source)
	}
	if err != nil {
		return nil, err
	}
	return event, nil
}

func (native *nativeService) copyRequest(source *C.coakka_http_host_request_event_t, exchange ExchangeID, handlerBindingID uint64) (*Request, error) {
	// One cgo crossing obtains Core's authoritative immutable view. The caller
	// still holds the event lease; all bytes become Go-owned before release.
	result := C.coakka_http_go_request_view(native.api, source)
	if err := resultError(result.status); err != nil {
		return nil, err
	}
	metadata := result.view
	method, err := nativeText(metadata.method)
	if err != nil {
		return nil, err
	}
	scheme, err := nativeText(metadata.scheme)
	if err != nil {
		return nil, err
	}
	authority, err := nativeText(metadata.authority)
	if err != nil {
		return nil, err
	}
	target, err := nativeText(metadata.target)
	if err != nil {
		return nil, err
	}
	body, err := nativeBytes(metadata.body)
	if err != nil {
		return nil, err
	}
	headers, err := native.copyRequestHeaders(source, metadata.header_count, 0)
	if err != nil {
		return nil, err
	}
	trailers, err := native.copyRequestHeaders(source, metadata.trailer_count, 1)
	if err != nil {
		return nil, err
	}
	pathParameters, err := native.copyPathParameters(source, metadata.path_parameter_count)
	if err != nil {
		return nil, err
	}
	queryParameters, err := native.copyQueryParameters(source, metadata.query_parameter_count)
	if err != nil {
		return nil, err
	}
	return &Request{
		Exchange:         exchange,
		Method:           method,
		Scheme:           scheme,
		Authority:        authority,
		Target:           target,
		Headers:          headers,
		PathParameters:   pathParameters,
		QueryParameters:  queryParameters,
		Trailers:         trailers,
		RouteID:          uint64(metadata.route_id),
		RouteGeneration:  uint64(metadata.route_generation),
		BindingRevision:  uint64(metadata.binding_revision),
		HandlerBindingID: handlerBindingID,
		BodyDelivery:     BodyDelivery(metadata.body_delivery),
		body:             body,
	}, nil
}

func checkedCount(value C.uint32_t) (int, error) {
	if uint64(value) > maximumProjectionItems {
		return 0, errors.New("native collection exceeds the Go bound")
	}
	return int(value), nil
}

func copiedHeader(source C.coakka_http_host_header_t) (Header, error) {
	name, err := nativeText(source.name)
	if err != nil {
		return Header{}, err
	}
	value, err := nativeText(source.value)
	if err != nil {
		return Header{}, err
	}
	// Core has already validated wire fields. Only memory-safe, bounded copying
	// belongs here; rescanning the same protocol bytes duplicates Core's work.
	return Header{Name: name, Value: value}, nil
}

func checkedHeaderCount(value C.uint32_t) (int, error) {
	if uint64(value) > maximumHeaders {
		return 0, errors.New("native header count exceeds the Go bound")
	}
	return int(value), nil
}

func (native *nativeService) copyRequestHeaders(event *C.coakka_http_host_request_event_t, countValue C.uint32_t, trailer uint8) (Headers, error) {
	count, err := checkedHeaderCount(countValue)
	if err != nil {
		return Headers{}, err
	}
	entries := make([]Header, 0, count)
	// Empty collections never enter C. Each nonempty bounded chunk is returned
	// by value, avoiding a heap-escaping Go output temporary for every entry.
	for index := 0; index < count; {
		size := min(count-index, int(C.COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY))
		var batch C.coakka_http_go_header_batch_t
		switch trailer {
		case 0:
			batch = C.coakka_http_go_request_headers(native.api, event, C.uint32_t(index), C.uint32_t(size))
		case 1:
			batch = C.coakka_http_go_request_trailers(native.api, event, C.uint32_t(index), C.uint32_t(size))
		case 2:
			batch = C.coakka_http_go_request_stream_trailers(native.api, event, C.uint32_t(index), C.uint32_t(size))
		}
		if batch.count != C.uint32_t(size) {
			return Headers{}, errors.New("native header projection failed")
		}
		for offset := 0; offset < size; offset++ {
			entry, copyErr := copiedHeader(batch.entries[offset])
			if copyErr != nil {
				return Headers{}, copyErr
			}
			entries = append(entries, entry)
		}
		index += size
	}
	return ownValidatedHeaders(entries), nil
}

func (native *nativeService) copyStreamTrailers(event *C.coakka_http_host_request_event_t) (Headers, error) {
	return native.copyRequestHeaders(event, C.coakka_http_go_request_stream_trailer_count(native.api, event), 2)
}

func (native *nativeService) copyPathParameters(event *C.coakka_http_host_request_event_t, countValue C.uint32_t) ([]PathParameter, error) {
	count, err := checkedCount(countValue)
	if err != nil {
		return nil, err
	}
	values := make([]PathParameter, 0, count)
	for index := 0; index < count; {
		size := min(count-index, int(C.COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY))
		batch := C.coakka_http_go_request_path_parameters(native.api, event, C.uint32_t(index), C.uint32_t(size))
		if batch.count != C.uint32_t(size) {
			return nil, errors.New("native path parameter projection failed")
		}
		for offset := 0; offset < size; offset++ {
			source := batch.entries[offset]
			name, copyErr := nativeText(source.name)
			if copyErr != nil {
				return nil, copyErr
			}
			value, copyErr := nativeText(source.encoded_value)
			if copyErr != nil {
				return nil, copyErr
			}
			values = append(values, PathParameter{Name: name, EncodedValue: value})
		}
		index += size
	}
	return values, nil
}

func (native *nativeService) copyQueryParameters(event *C.coakka_http_host_request_event_t, countValue C.uint32_t) ([]QueryParameter, error) {
	count, err := checkedCount(countValue)
	if err != nil {
		return nil, err
	}
	values := make([]QueryParameter, 0, count)
	for index := 0; index < count; {
		size := min(count-index, int(C.COAKKA_HTTP_GO_ENTRY_BATCH_CAPACITY))
		batch := C.coakka_http_go_request_query_parameters(native.api, event, C.uint32_t(index), C.uint32_t(size))
		if batch.count != C.uint32_t(size) {
			return nil, errors.New("native query parameter projection failed")
		}
		for offset := 0; offset < size; offset++ {
			source := batch.entries[offset]
			key, copyErr := nativeText(source.encoded_key)
			if copyErr != nil {
				return nil, copyErr
			}
			value, copyErr := nativeText(source.encoded_value)
			if copyErr != nil {
				return nil, copyErr
			}
			values = append(values, QueryParameter{EncodedKey: key, EncodedValue: value, HasValue: source.has_value != 0})
		}
		index += size
	}
	return values, nil
}

func (native *nativeService) exchangeCancelled(exchange ExchangeID) (bool, error) {
	var cancelled C.uint8_t
	err := resultError(C.coakka_http_go_request_cancelled(native.api, native.service, nativeExchange(exchange), &cancelled))
	return cancelled != 0, err
}

func (native *nativeService) respond(exchange ExchangeID, response Response) error {
	if response.Status < 100 || response.Status > 599 {
		return errors.New("HTTP response status is invalid")
	}
	if len(response.body) > maximumBridgeBytes {
		return errors.New("native byte projection exceeds the Go bound")
	}
	if response.buffered.matches(response.Headers) {
		plan := response.buffered
		var wire *C.uint8_t
		if len(plan.wire) != 0 {
			wire = (*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(plan.wire)))
		}
		status := C.coakka_http_go_respond_prepared(
			native.api, native.service, nativeExchange(exchange),
			C.uint32_t(response.Status), wire, C.size_t(len(plan.wire)),
			C.uint32_t(plan.headerCount), C.size_t(plan.bodyOffset),
		)
		runtime.KeepAlive(plan)
		return resultError(status)
	}
	arena := nativeArena{}
	defer arena.close()
	headers, count, err := arena.headers(response.Headers)
	if err != nil {
		return err
	}
	var body *C.uint8_t
	if len(response.body) != 0 {
		body = (*C.uint8_t)(unsafe.Pointer(unsafe.SliceData(response.body)))
	}
	// cgo pins this pointer-free backing array for the synchronous call. The
	// bridge builds a C-stack view; Core copies accepted bytes into its bounded
	// queue before returning. No Go slice header or Go pointer is retained by C.
	// Keep NewResponse's defensive copy; only C staging is removed here.
	status := C.coakka_http_go_respond_buffered(native.api, native.service,
		nativeExchange(exchange), C.uint32_t(response.Status), headers, count,
		body, C.size_t(len(response.body)))
	runtime.KeepAlive(response.body)
	return resultError(status)
}

func (native *nativeService) respondFile(exchange ExchangeID, response FileResponse) error {
	arena := nativeArena{}
	defer arena.close()
	var value C.coakka_http_host_file_response_t
	C.coakka_http_go_file_response_init(native.api, &value)
	head, err := nativeResponse(native.api, &arena, response.Head)
	if err != nil {
		return err
	}
	value.head = head
	value.authority_id = C.uint64_t(response.AuthorityID)
	if value.encoded_path, err = arena.text(response.EncodedPath); err != nil {
		return err
	}
	return resultError(C.coakka_http_go_respond_file(native.api, native.service, nativeExchange(exchange), &value))
}

func (native *nativeService) fail(exchange ExchangeID, failure Failure) error {
	arena := nativeArena{}
	defer arena.close()
	var value C.coakka_http_host_failure_t
	C.coakka_http_go_failure_init(native.api, &value)
	value.reason = C.uint32_t(failure.Reason)
	var err error
	if value.detail, err = arena.text(failure.Detail); err != nil {
		return err
	}
	return resultError(C.coakka_http_go_fail(native.api, native.service, nativeExchange(exchange), &value))
}

func (native *nativeService) startResponseStream(exchange ExchangeID, head Response) error {
	arena := nativeArena{}
	defer arena.close()
	value, err := nativeResponse(native.api, &arena, head)
	if err != nil {
		return err
	}
	return resultError(C.coakka_http_go_start_response_stream(native.api, native.service, nativeExchange(exchange), &value))
}

func (native *nativeService) startSSE(exchange ExchangeID, headers Headers) error {
	arena := nativeArena{}
	defer arena.close()
	values, count, err := arena.headers(headers)
	if err != nil {
		return err
	}
	return resultError(C.coakka_http_go_start_sse(native.api, native.service, nativeExchange(exchange), values, count))
}

func (native *nativeService) writeResponseStream(exchange ExchangeID, data []byte) error {
	arena := nativeArena{}
	defer arena.close()
	value, err := arena.bytes(data)
	if err != nil {
		return err
	}
	return resultError(C.coakka_http_go_write_response_stream(native.api, native.service, nativeExchange(exchange), value))
}

func (native *nativeService) writeSSE(exchange ExchangeID, event SSEEvent) error {
	arena := nativeArena{}
	defer arena.close()
	var value C.coakka_http_host_sse_event_t
	C.coakka_http_go_sse_event_init(native.api, &value)
	var err error
	if value.data, err = arena.bytes(event.Data); err != nil {
		return err
	}
	if value.event_type, err = arena.text(event.EventType); err != nil {
		return err
	}
	if value.id, err = arena.text(event.ID); err != nil {
		return err
	}
	value.retry_ms = C.uint64_t(event.RetryMillis)
	value.has_event_type = boolByte(event.HasEventType)
	value.has_id = boolByte(event.HasID)
	value.has_retry = boolByte(event.HasRetry)
	return resultError(C.coakka_http_go_write_sse(native.api, native.service, nativeExchange(exchange), &value))
}

func (native *nativeService) finishResponseStream(exchange ExchangeID, trailers Headers) error {
	arena := nativeArena{}
	defer arena.close()
	values, count, err := arena.headers(trailers)
	if err != nil {
		return err
	}
	return resultError(C.coakka_http_go_finish_response_stream(native.api, native.service, nativeExchange(exchange), values, count))
}
