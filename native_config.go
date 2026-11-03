package coakkahttp

/*
#include "bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"
)

func openNativeService(config Config) (_ *nativeService, returned error) {
	// The public Go enum becomes a native boolean. Reject values which cannot
	// be represented rather than silently mapping an unknown mode to disabled.
	// Numeric compression policy and capability checks remain Core-owned.
	if config.Compression != nil && config.Compression.Mode != 0 &&
		config.Compression.Mode != CompressionDisabled && config.Compression.Mode != CompressionGZIP {
		return nil, errors.New("compression mode cannot be represented by the Core configuration")
	}
	api, err := openNativeAPI()
	if err != nil {
		return nil, err
	}
	defer func() {
		if returned != nil {
			C.coakka_http_go_close(&api)
		}
	}()
	if len(config.Listeners) > 1 {
		return nil, errors.New("the native service accepts exactly one listener")
	}
	arena := nativeArena{}
	defer arena.close()
	var configuration C.coakka_http_host_configuration_t
	C.coakka_http_go_configuration_init(api, &configuration)
	if config.Limits != nil {
		configuration.limits = nativeLimits(api, *config.Limits)
	}
	listener := Listener{}
	if len(config.Listeners) == 1 {
		listener = config.Listeners[0]
	}
	if err = projectListener(&arena, &configuration, listener); err != nil {
		return nil, err
	}
	// Opt-in is deliberately connector-visible, while support probing and the
	// Linux fallback remain entirely native-owned.
	configuration.use_io_uring = boolByte(config.IOBackend == IOUring)

	if config.Monitor != nil {
		configuration.monitor = nativeMonitorOptions(api, *config.Monitor)
	}
	if config.Compression != nil {
		configuration.compression = nativeCompression(api, *config.Compression)
	}
	if err = projectStaticMounts(api, &arena, &configuration, config.StaticMounts); err != nil {
		return nil, err
	}
	if err = projectFileAuthorities(api, &arena, &configuration, config.FileAuthorities); err != nil {
		return nil, err
	}
	if err = projectOutbound(api, &arena, &configuration, config); err != nil {
		return nil, err
	}

	routes, routeCount, err := projectRoutes(api, &arena, config.Routes)
	if err != nil {
		return nil, err
	}
	var inspection *C.coakka_http_host_inspection_options_t
	var inspectionValue C.coakka_http_host_inspection_options_t
	if options := config.Inspection; options != nil {
		C.coakka_http_host_inspection_options_init(&inspectionValue)
		inspectionValue.runtime_instance_id = C.uint64_t(options.RuntimeInstanceID)
		inspectionValue.runtime_generation = C.uint64_t(options.RuntimeGeneration)
		inspectionValue.allow_plaintext_loopback = boolByte(options.AllowPlaintextLoopback)
		inspectionValue.bearer_file, err = arena.text(options.BearerFile)
		if err != nil {
			return nil, err
		}
		inspection = &inspectionValue
	}
	var service *C.coakka_http_host_service_t
	failedIndex := C.uint32_t(C.COAKKA_HTTP_HOST_FAILED_INDEX_NONE)
	if err = resultError(C.coakka_http_go_service_create(
		api, &configuration, inspection, routes, routeCount, &service, &failedIndex)); err != nil {
		if uint32(failedIndex) != uint32(C.COAKKA_HTTP_HOST_FAILED_INDEX_NONE) {
			return nil, fmt.Errorf("route %d: %w", uint32(failedIndex), err)
		}
		return nil, err
	}
	if service == nil {
		return nil, errors.New("native library returned an empty service")
	}
	return &nativeService{api: api, service: service, capabilities: nativeFeatures(api)}, nil
}

func nativeLimits(api *C.coakka_http_go_api_t, source RuntimeLimits) C.coakka_http_host_limits_t {
	var value C.coakka_http_host_limits_t
	C.coakka_http_go_limits_init(api, &value)
	set32 := func(target *C.uint32_t, source uint32) {
		if source != 0 {
			*target = C.uint32_t(source)
		}
	}
	set64 := func(target *C.uint64_t, source uint64) {
		if source != 0 {
			*target = C.uint64_t(source)
		}
	}
	set32(&value.max_connections, source.MaxConnections)
	set32(&value.max_active_requests, source.MaxActiveExchanges)
	set32(&value.request_queue_capacity, source.RequestQueueCapacity)
	set32(&value.completion_queue_capacity, source.CompletionQueueCapacity)
	set32(&value.completion_batch_size, source.CompletionBatchSize)
	set32(&value.max_header_count, source.MaxHeaderCount)
	set32(&value.max_header_bytes, source.MaxHeaderBytes)
	set32(&value.max_routes, source.MaxRoutes)
	set32(&value.max_total_segments, source.MaxTotalRouteSegments)
	set32(&value.max_route_bytes, source.MaxRouteBytes)
	set64(&value.max_total_route_metadata_bytes, source.MaxTotalRouteMetadataBytes)
	set32(&value.max_captures_per_route, source.MaxCapturesPerRoute)
	set32(&value.activation_history_capacity, source.ActivationHistoryCapacity)
	set32(&value.request_stream_chunk_slots, source.RequestStreamChunkSlots)
	set32(&value.request_stream_max_retained_items, source.RequestStreamMaxRetainedItems)
	set32(&value.response_stream_chunk_slots, source.ResponseStreamChunkSlots)
	set32(&value.response_stream_max_retained_items, source.ResponseStreamMaxRetainedItems)
	set32(&value.route_control_capacity, source.RouteControlCapacity)
	set32(&value.route_control_history_capacity, source.RouteControlHistoryCapacity)
	set32(&value.route_control_max_commands_per_turn, source.RouteControlMaxCommandsPerTurn)
	set32(&value.terminal_queue_capacity, source.TerminalQueueCapacity)
	set32(&value.deadline_tick_ms, source.DeadlineTickMillis)
	set32(&value.connection_timeout_batch_size, source.ConnectionTimeoutBatchSize)
	value.request_notification_profile = C.coakka_http_host_notification_profile_t(source.Notifications.Request)
	value.terminal_notification_profile = C.coakka_http_host_notification_profile_t(source.Notifications.Terminal)
	set64(&value.event_loop_cpu_mask, source.EventLoopCPUMask)
	set64(&value.max_request_body_bytes, source.MaxRequestBodyBytes)
	set64(&value.max_response_body_bytes, source.MaxResponseBodyBytes)
	set64(&value.keep_alive_timeout_ms, source.KeepAliveTimeoutMillis)
	set64(&value.header_timeout_ms, source.HeaderTimeoutMillis)
	set64(&value.body_timeout_ms, source.BodyTimeoutMillis)
	set64(&value.handler_timeout_ms, source.AppHostTimeoutMillis)
	set64(&value.idle_timeout_ms, source.IdleTimeoutMillis)
	set64(&value.drain_timeout_ms, source.DrainTimeoutMillis)
	set64(&value.stop_timeout_ms, source.StopTimeoutMillis)
	set64(&value.response_write_timeout_ms, source.ResponseWriteTimeoutMillis)
	set64(&value.request_stream_max_chunk_bytes, source.RequestStreamMaxChunkBytes)
	set64(&value.request_stream_max_trailer_bytes, source.RequestStreamMaxTrailerBytes)
	set64(&value.request_stream_max_retained_bytes, source.RequestStreamMaxRetainedBytes)
	set64(&value.request_stream_max_total_retained_bytes, source.RequestStreamMaxTotalRetainedBytes)
	set64(&value.response_stream_max_chunk_bytes, source.ResponseStreamMaxChunkBytes)
	set64(&value.response_stream_max_trailer_bytes, source.ResponseStreamMaxTrailerBytes)
	set64(&value.response_stream_max_retained_bytes, source.ResponseStreamMaxRetainedBytes)
	set64(&value.response_stream_max_total_retained_bytes, source.ResponseStreamMaxTotalRetainedBytes)
	return value
}

// projectEffectiveLimits is a value projection, never a second default table.
// The initialized Core service has already resolved and validated every field.
func projectEffectiveLimits(value C.coakka_http_host_limits_t) RuntimeLimits {
	return RuntimeLimits{
		MaxRoutes:                           uint32(value.max_routes),
		MaxTotalRouteSegments:               uint32(value.max_total_segments),
		MaxRouteBytes:                       uint32(value.max_route_bytes),
		MaxTotalRouteMetadataBytes:          uint64(value.max_total_route_metadata_bytes),
		MaxCapturesPerRoute:                 uint32(value.max_captures_per_route),
		ActivationHistoryCapacity:           uint32(value.activation_history_capacity),
		MaxConnections:                      uint32(value.max_connections),
		MaxActiveExchanges:                  uint32(value.max_active_requests),
		RequestQueueCapacity:                uint32(value.request_queue_capacity),
		CompletionQueueCapacity:             uint32(value.completion_queue_capacity),
		CompletionBatchSize:                 uint32(value.completion_batch_size),
		MaxHeaderCount:                      uint32(value.max_header_count),
		MaxHeaderBytes:                      uint32(value.max_header_bytes),
		RequestStreamChunkSlots:             uint32(value.request_stream_chunk_slots),
		RequestStreamMaxRetainedItems:       uint32(value.request_stream_max_retained_items),
		ResponseStreamChunkSlots:            uint32(value.response_stream_chunk_slots),
		ResponseStreamMaxRetainedItems:      uint32(value.response_stream_max_retained_items),
		RouteControlCapacity:                uint32(value.route_control_capacity),
		RouteControlHistoryCapacity:         uint32(value.route_control_history_capacity),
		RouteControlMaxCommandsPerTurn:      uint32(value.route_control_max_commands_per_turn),
		TerminalQueueCapacity:               uint32(value.terminal_queue_capacity),
		DeadlineTickMillis:                  uint32(value.deadline_tick_ms),
		ConnectionTimeoutBatchSize:          uint32(value.connection_timeout_batch_size),
		Notifications:                       NotificationProfiles{Request: NotificationProfile(value.request_notification_profile), Terminal: NotificationProfile(value.terminal_notification_profile)},
		EventLoopCPUMask:                    uint64(value.event_loop_cpu_mask),
		MaxRequestBodyBytes:                 uint64(value.max_request_body_bytes),
		MaxResponseBodyBytes:                uint64(value.max_response_body_bytes),
		KeepAliveTimeoutMillis:              uint64(value.keep_alive_timeout_ms),
		HeaderTimeoutMillis:                 uint64(value.header_timeout_ms),
		BodyTimeoutMillis:                   uint64(value.body_timeout_ms),
		AppHostTimeoutMillis:                uint64(value.handler_timeout_ms),
		IdleTimeoutMillis:                   uint64(value.idle_timeout_ms),
		DrainTimeoutMillis:                  uint64(value.drain_timeout_ms),
		StopTimeoutMillis:                   uint64(value.stop_timeout_ms),
		ResponseWriteTimeoutMillis:          uint64(value.response_write_timeout_ms),
		RequestStreamMaxChunkBytes:          uint64(value.request_stream_max_chunk_bytes),
		RequestStreamMaxTrailerBytes:        uint64(value.request_stream_max_trailer_bytes),
		RequestStreamMaxRetainedBytes:       uint64(value.request_stream_max_retained_bytes),
		RequestStreamMaxTotalRetainedBytes:  uint64(value.request_stream_max_total_retained_bytes),
		ResponseStreamMaxChunkBytes:         uint64(value.response_stream_max_chunk_bytes),
		ResponseStreamMaxTrailerBytes:       uint64(value.response_stream_max_trailer_bytes),
		ResponseStreamMaxRetainedBytes:      uint64(value.response_stream_max_retained_bytes),
		ResponseStreamMaxTotalRetainedBytes: uint64(value.response_stream_max_total_retained_bytes),
	}
}

func projectListener(arena *nativeArena, value *C.coakka_http_host_configuration_t, source Listener) error {
	value.port = C.uint16_t(source.Port)
	// The Core initializer owns absent protocol/security defaults; an empty
	// bind address is passed through for Core's loopback default resolution.
	if source.Protocol != 0 {
		value.protocol = C.coakka_http_host_protocol_t(source.Protocol)
	}
	if source.Security != 0 {
		value.security = C.coakka_http_host_security_t(source.Security)
	}
	value.credential_generation = C.uint64_t(source.CredentialGeneration)
	var err error
	if value.bind_address, err = arena.text(source.BindAddress); err != nil {
		return err
	}
	if value.credential_id, err = arena.text(source.CredentialID); err != nil {
		return err
	}
	if value.certificate_chain_file, err = arena.text(source.CertificateChainFile); err != nil {
		return err
	}
	if value.private_key_file, err = arena.text(source.PrivateKeyFile); err != nil {
		return err
	}
	if value.trust_roots_file, err = arena.text(source.TrustRootsFile); err != nil {
		return err
	}
	return nil
}

func projectRoutes(api *C.coakka_http_go_api_t, arena *nativeArena, source []Route) (*C.coakka_http_host_route_t, C.uint32_t, error) {
	if len(source) == 0 {
		return nil, 0, nil
	}
	if len(source) > maximumProjectionItems {
		return nil, 0, errors.New("route count exceeds the Go projection bound")
	}
	memory, err := arena.allocate(uintptr(len(source)) * unsafe.Sizeof(C.coakka_http_host_route_t{}))
	if err != nil {
		return nil, 0, err
	}
	values := unsafe.Slice((*C.coakka_http_host_route_t)(memory), len(source))
	metadata := nativeMetadataProjection{api: api, arena: arena}
	for index := range source {
		if err = projectNativeRoute(api, arena, &metadata, source[index], &values[index]); err != nil {
			return nil, 0, fmt.Errorf("route %d: %w", index, err)
		}
	}
	return (*C.coakka_http_host_route_t)(memory), C.uint32_t(len(values)), nil
}

func projectNativeRoute(api *C.coakka_http_go_api_t, arena *nativeArena,
	metadata *nativeMetadataProjection, source Route, value *C.coakka_http_host_route_t) error {
	// This Go enum projects to a Core flag, not a wire enum. Reject values that
	// cannot be represented instead of silently treating them as omission.
	if source.BodyDelivery != 0 && source.BodyDelivery != BodyInline && source.BodyDelivery != BodyStream {
		return errors.New("request body delivery cannot be represented by the native route contract")
	}
	C.coakka_http_go_route_init(api, value)
	value.route_id = C.uint64_t(source.ID)
	value.handler_binding_id = C.uint64_t(source.HandlerBindingID)
	if source.BodyDelivery == BodyStream {
		value.flags |= C.COAKKA_HTTP_HOST_ROUTE_STREAMING_REQUEST
	}
	if source.WebSocket {
		value.flags |= C.COAKKA_HTTP_HOST_ROUTE_WEBSOCKET
	}
	value.body_policy.enabled = boolByte(source.BodyPolicy.Enabled)
	value.body_policy.accept_absent = boolByte(source.BodyPolicy.AcceptAbsent)
	value.body_policy.accept_other = boolByte(source.BodyPolicy.AcceptOther)
	value.body_policy.accept_form_urlencoded = boolByte(source.BodyPolicy.AcceptFormURLEncoded)
	value.body_policy.accept_multipart_form_data = boolByte(source.BodyPolicy.AcceptMultipartFormData)
	value.body_policy.max_body_bytes = C.uint64_t(source.BodyPolicy.MaxBodyBytes)
	var err error
	if value.method, err = arena.text(source.Method); err != nil {
		return err
	}
	if value.encoded_path_pattern, err = arena.text(source.Pattern); err != nil {
		return err
	}
	value.api_metadata, err = metadata.route(source.APIMetadata)
	return err
}

func projectStaticMounts(api *C.coakka_http_go_api_t, arena *nativeArena, configuration *C.coakka_http_host_configuration_t, source []StaticMount) error {
	if len(source) == 0 {
		return nil
	}
	memory, err := arena.allocate(uintptr(len(source)) * unsafe.Sizeof(C.coakka_http_host_static_mount_t{}))
	if err != nil {
		return err
	}
	values := unsafe.Slice((*C.coakka_http_host_static_mount_t)(memory), len(source))
	for index := range source {
		item := source[index]
		value := &values[index]
		C.coakka_http_go_static_mount_init(api, value)
		if item.IncludeDotfiles {
			value.flags |= C.COAKKA_HTTP_HOST_STATIC_INCLUDE_DOTFILES
		}
		if item.HasSPAFallbackFile {
			value.flags |= C.COAKKA_HTTP_HOST_STATIC_SPA_FALLBACK
		}
		if value.url_prefix, err = arena.text(item.URLPrefix); err != nil {
			return err
		}
		if value.root_path, err = arena.text(item.RootPath); err != nil {
			return err
		}
		if item.HasIndexFile {
			if value.index_file, err = arena.text(item.IndexFile); err != nil {
				return err
			}
		}
		if item.HasSPAFallbackFile {
			if value.spa_fallback_file, err = arena.text(item.SPAFallbackFile); err != nil {
				return err
			}
		}
		if item.HasCacheControl {
			if value.cache_control, err = arena.text(item.CacheControl); err != nil {
				return err
			}
		}
	}
	configuration.static_mounts = (*C.coakka_http_host_static_mount_t)(memory)
	configuration.static_mount_count = C.uint32_t(len(values))
	return nil
}

func projectFileAuthorities(api *C.coakka_http_go_api_t, arena *nativeArena, configuration *C.coakka_http_host_configuration_t, source []FileAuthority) error {
	if len(source) == 0 {
		return nil
	}
	memory, err := arena.allocate(uintptr(len(source)) * unsafe.Sizeof(C.coakka_http_host_file_authority_t{}))
	if err != nil {
		return err
	}
	values := unsafe.Slice((*C.coakka_http_host_file_authority_t)(memory), len(source))
	for index := range source {
		item := source[index]
		value := &values[index]
		C.coakka_http_go_file_authority_init(api, value)
		value.authority_id = C.uint64_t(item.ID)
		value.max_active_files = C.uint32_t(item.MaxActiveFiles)
		value.max_file_bytes = C.uint64_t(item.MaxFileBytes)
		if value.root_path, err = arena.text(item.RootPath); err != nil {
			return err
		}
	}
	configuration.file_authorities = (*C.coakka_http_host_file_authority_t)(memory)
	configuration.file_authority_count = C.uint32_t(len(values))
	return nil
}

func nativeCompression(api *C.coakka_http_go_api_t, source Compression) C.coakka_http_host_compression_t {
	var value C.coakka_http_host_compression_t
	C.coakka_http_go_compression_init(api, &value)
	value.enabled = boolByte(source.Mode == CompressionGZIP)
	value.minimum_body_bytes = C.uint64_t(source.MinimumBodyBytes)
	value.maximum_body_bytes = C.uint64_t(source.MaximumBodyBytes)
	value.maximum_encoded_bytes = C.uint64_t(source.MaximumEncodedBytes)
	value.workspace_bytes = C.uint64_t(source.WorkspaceBytes)
	value.gzip_level = C.int32_t(source.GZIPLevel)
	return value
}

func nativeMonitorCategories(source MonitorCategory) C.coakka_http_host_monitor_category_t {
	return C.coakka_http_host_monitor_category_t(source)
}

func nativeMonitorOptions(api *C.coakka_http_go_api_t, source MonitorOptions) C.coakka_http_host_monitor_options_t {
	var value C.coakka_http_host_monitor_options_t
	C.coakka_http_go_monitor_options_init(api, &value)
	if source.EventCapacity != 0 {
		value.event_capacity = C.uint32_t(source.EventCapacity)
	}
	if source.MaxEventsPerRead != 0 {
		value.max_events_per_read = C.uint32_t(source.MaxEventsPerRead)
	}
	if categories := nativeMonitorCategories(source.AggregateCategories | source.EventCategories); categories != 0 {
		value.categories = categories
	}
	value.initial_collection = C.coakka_http_host_monitor_collection_t(source.Collection)
	value.signal_reserved = boolByte(source.SignalReserved)
	value.fixed_latency_buckets = boolByte(source.FixedLatencyBuckets)
	return value
}

func projectOutbound(api *C.coakka_http_go_api_t, arena *nativeArena, configuration *C.coakka_http_host_configuration_t, config Config) error {
	if err := projectOutboundTrust(api, arena, configuration, config.OutboundTrust); err != nil {
		return err
	}
	if err := projectOutboundIdentities(api, arena, configuration, config.OutboundIdentities); err != nil {
		return err
	}
	if len(config.OutboundTargets) == 0 {
		return nil
	}
	memory, err := arena.allocate(uintptr(len(config.OutboundTargets)) * unsafe.Sizeof(C.coakka_http_host_outbound_target_t{}))
	if err != nil {
		return err
	}
	values := unsafe.Slice((*C.coakka_http_host_outbound_target_t)(memory), len(config.OutboundTargets))
	for index := range config.OutboundTargets {
		source := config.OutboundTargets[index]
		value := &values[index]
		C.coakka_http_go_outbound_target_init(api, value)
		if source.Strategy != 0 {
			value.strategy = C.coakka_http_host_outbound_strategy_t(source.Strategy)
		}
		value.generation = C.uint64_t(source.Generation)
		if value.name, err = arena.text(source.Name); err != nil {
			return err
		}
		if len(source.Endpoints) == 0 {
			continue
		}
		endpointMemory, allocationErr := arena.allocate(uintptr(len(source.Endpoints)) * unsafe.Sizeof(C.coakka_http_host_outbound_endpoint_t{}))
		if allocationErr != nil {
			return allocationErr
		}
		endpoints := unsafe.Slice((*C.coakka_http_host_outbound_endpoint_t)(endpointMemory), len(source.Endpoints))
		for endpointIndex := range source.Endpoints {
			item := source.Endpoints[endpointIndex]
			endpoint := &endpoints[endpointIndex]
			C.coakka_http_go_outbound_endpoint_init(api, endpoint)
			if item.Weight != 0 {
				endpoint.weight = C.uint32_t(item.Weight)
			}
			endpoint.connect_port = C.uint16_t(item.ConnectPort)
			if item.Security != 0 {
				endpoint.security = C.coakka_http_host_security_t(item.Security)
			}
			if item.Available != nil {
				endpoint.available = C.uint32_t(boolByte(*item.Available))
			}
			endpoint.tls_trust_generation = C.uint64_t(item.TLSTrustGeneration)
			endpoint.tls_client_identity_generation = C.uint64_t(item.TLSClientIdentityGeneration)
			if endpoint.node_id, err = arena.text(item.NodeID); err != nil {
				return err
			}
			if endpoint.connect_host, err = arena.text(item.ConnectHost); err != nil {
				return err
			}
			if endpoint.http_authority, err = arena.text(item.HTTPAuthority); err != nil {
				return err
			}
			if endpoint.tls_peer_identity, err = arena.text(item.TLSPeerIdentity); err != nil {
				return err
			}
		}
		value.endpoints = (*C.coakka_http_host_outbound_endpoint_t)(endpointMemory)
		value.endpoint_count = C.uint32_t(len(endpoints))
	}
	configuration.outbound_targets = (*C.coakka_http_host_outbound_target_t)(memory)
	configuration.outbound_target_count = C.uint32_t(len(values))
	return nil
}

func projectOutboundTrust(api *C.coakka_http_go_api_t, arena *nativeArena, configuration *C.coakka_http_host_configuration_t, source []OutboundTrust) error {
	if len(source) == 0 {
		return nil
	}
	memory, err := arena.allocate(uintptr(len(source)) * unsafe.Sizeof(C.coakka_http_host_outbound_trust_t{}))
	if err != nil {
		return err
	}
	values := unsafe.Slice((*C.coakka_http_host_outbound_trust_t)(memory), len(source))
	for index := range source {
		C.coakka_http_go_outbound_trust_init(api, &values[index])
		values[index].generation = C.uint64_t(source[index].Generation)
		if values[index].ca_pem, err = arena.bytes(source[index].CAPEM); err != nil {
			return err
		}
	}
	configuration.outbound_trusts = (*C.coakka_http_host_outbound_trust_t)(memory)
	configuration.outbound_trust_count = C.uint32_t(len(values))
	return nil
}

func projectOutboundIdentities(api *C.coakka_http_go_api_t, arena *nativeArena, configuration *C.coakka_http_host_configuration_t, source []OutboundIdentity) error {
	if len(source) == 0 {
		return nil
	}
	memory, err := arena.allocate(uintptr(len(source)) * unsafe.Sizeof(C.coakka_http_host_outbound_identity_t{}))
	if err != nil {
		return err
	}
	values := unsafe.Slice((*C.coakka_http_host_outbound_identity_t)(memory), len(source))
	for index := range source {
		C.coakka_http_go_outbound_identity_init(api, &values[index])
		values[index].generation = C.uint64_t(source[index].Generation)
		if values[index].certificate_chain_pem, err = arena.bytes(source[index].CertificateChainPEM); err != nil {
			return err
		}
		if values[index].private_key_pem, err = arena.bytes(source[index].PrivateKeyPEM); err != nil {
			return err
		}
	}
	configuration.outbound_identities = (*C.coakka_http_host_outbound_identity_t)(memory)
	configuration.outbound_identity_count = C.uint32_t(len(values))
	return nil
}
