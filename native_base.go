package coakkahttp

/*
#cgo CFLAGS: -std=c11 -Wall -Wextra -Wpedantic -Wconversion -Wsign-conversion -Werror
// Go 1.23's generated _cgo_main.c uses an old-style main declaration. AppleClang
// 17 and LLVM-MinGW Clang 23 diagnose that generated declaration; owned bridge
// code keeps full C11 gates.
#cgo darwin CFLAGS: -Wno-strict-prototypes
#cgo windows CFLAGS: -Wno-strict-prototypes
#cgo LDFLAGS: -lcoakka_http_host
#include "bridge.h"
#include <stdlib.h>
*/
import "C"

import (
	"errors"
	"fmt"
	"time"
	"unsafe"
)

const (
	maximumBridgeBytes     = 64 << 20
	maximumProjectionItems = 1 << 20
	maximumNativeWait      = 30 * time.Second
)

// nativeService is deliberately small: it owns the installed native service and
// no connector-side transport, platform probe, or copied dispatch table.
type nativeService struct {
	api          *C.coakka_http_go_api_t
	service      *C.coakka_http_host_service_t
	capabilities Capabilities
}

type nativeArena struct {
	allocations []unsafe.Pointer
}

func (arena *nativeArena) close() {
	for index := len(arena.allocations) - 1; index >= 0; index-- {
		C.free(arena.allocations[index])
	}
	arena.allocations = nil
}

func (arena *nativeArena) allocate(size uintptr) (unsafe.Pointer, error) {
	if size == 0 {
		return nil, nil
	}
	pointer := C.calloc(1, C.size_t(size))
	if pointer == nil {
		return nil, errors.New("native projection allocation failed")
	}
	arena.allocations = append(arena.allocations, pointer)
	return pointer, nil
}

func (arena *nativeArena) bytes(value []byte) (C.coakka_http_host_bytes_t, error) {
	if len(value) == 0 {
		return C.coakka_http_host_bytes_t{}, nil
	}
	if len(value) > maximumBridgeBytes {
		return C.coakka_http_host_bytes_t{}, errors.New("native byte projection exceeds the Go bound")
	}
	pointer := C.CBytes(value)
	if pointer == nil {
		return C.coakka_http_host_bytes_t{}, errors.New("native byte projection allocation failed")
	}
	arena.allocations = append(arena.allocations, pointer)
	return C.coakka_http_host_bytes_t{data: (*C.uint8_t)(pointer), size: C.size_t(len(value))}, nil
}

func (arena *nativeArena) text(value string) (C.coakka_http_host_bytes_t, error) {
	return arena.bytes([]byte(value))
}

func boolByte(value bool) C.uint8_t {
	if value {
		return 1
	}
	return 0
}

func nativeDuration(timeout time.Duration) (C.uint64_t, error) {
	if timeout < 0 {
		return 0, errors.New("timeout must not be negative")
	}
	if timeout > maximumNativeWait {
		return 0, fmt.Errorf("timeout exceeds the bounded maximum %s", maximumNativeWait)
	}
	if timeout == 0 {
		return 0, nil
	}
	milliseconds := (timeout + time.Millisecond - 1) / time.Millisecond
	return C.uint64_t(milliseconds), nil
}

func resultError(status C.coakka_http_host_status_t) error {
	if status == C.COAKKA_HTTP_HOST_OK {
		return nil
	}
	return &Error{
		Code:   ResultCode(status),
		Detail: C.GoString(C.coakka_http_host_status_name(status)),
	}
}

func issueError(status C.coakka_http_host_status_t, issue *C.coakka_http_host_issue_t) error {
	if status == C.COAKKA_HTTP_HOST_OK {
		return nil
	}
	if issue == nil {
		return resultError(status)
	}
	detail := C.GoString((*C.char)(unsafe.Pointer(&issue.detail[0])))
	if detail == "" {
		detail = C.GoString(C.coakka_http_host_status_name(status))
	}
	base := resultError(status).(*Error)
	base.Reason = FailureReason(issue.reason)
	base.SystemCode = int32(issue.system_code)
	base.Actual = uint64(issue.actual)
	base.Limit = uint64(issue.limit)
	base.Detail = detail
	return base
}

func nativePlanService(api *C.coakka_http_go_api_t, required, adapter Capabilities) (ServicePlan, error) {
	var request C.coakka_http_host_service_plan_request_t
	var plan C.coakka_http_host_service_plan_t
	C.coakka_http_go_service_plan_request_init(api, &request)
	C.coakka_http_go_service_plan_init(api, &plan)
	request.required_capabilities = C.coakka_http_host_capabilities_t(required)
	request.adapter_capabilities = C.coakka_http_host_capabilities_t(adapter)
	status := C.coakka_http_go_plan_service(api, &request, &plan)
	if status != C.COAKKA_HTTP_HOST_OK && status != C.COAKKA_HTTP_HOST_UNSUPPORTED {
		return ServicePlan{}, resultError(status)
	}
	return ServicePlan{
		Supported:            plan.supported != 0,
		RequiredCapabilities: Capabilities(plan.required_capabilities),
		AdapterCapabilities:  Capabilities(plan.adapter_capabilities),
		MissingCapabilities:  Capabilities(plan.missing_capabilities),
	}, nil
}

func nativeBytes(value C.coakka_http_host_bytes_t) ([]byte, error) {
	if value.size == 0 {
		return nil, nil
	}
	if value.data == nil || uint64(value.size) > maximumBridgeBytes {
		return nil, errors.New("native byte range is outside the Go bound")
	}
	return C.GoBytes(unsafe.Pointer(value.data), C.int(value.size)), nil
}

func nativeText(value C.coakka_http_host_bytes_t) (string, error) {
	if value.size == 0 {
		return "", nil
	}
	if value.data == nil || uint64(value.size) > maximumBridgeBytes {
		return "", errors.New("native byte range is outside the Go bound")
	}
	// Copy directly into the final immutable Go string. A GoBytes intermediate
	// used to allocate/copy the same native range twice. The lease still owns
	// the source for this call; no unsafe alias survives its release.
	return C.GoStringN((*C.char)(unsafe.Pointer(value.data)), C.int(value.size)), nil
}

func openNativeAPI() (*C.coakka_http_go_api_t, error) {
	var api *C.coakka_http_go_api_t
	if err := resultError(C.coakka_http_go_open(&api)); err != nil {
		return nil, err
	}
	if api == nil {
		return nil, errors.New("native library returned an empty handle")
	}
	return api, nil
}

func nativeFeatures(api *C.coakka_http_go_api_t) Capabilities {
	return Capabilities(C.coakka_http_go_features(api))
}

func projectRuntimeInfo(value C.coakka_http_host_service_info_t, features Capabilities) RuntimeInfo {
	requested := IOPlatformDefault
	if value.io_uring_requested != 0 {
		requested = IOUring
	}
	effective := IOPlatformDefault
	if value.io_uring_effective != 0 {
		effective = IOUring
	}
	return RuntimeInfo{
		Execution: ExecutionInfo{
			Selection: ExecutionSelection{
				RequestedMode:  ExecutionMode(value.selection.requested_mode),
				RequestedCount: uint32(value.selection.requested_count),
				SelectedCount:  uint32(value.selection.selected_count),
				Reason:         ExecutionReason(value.selection.reason),
			},
			ConfiguredEventLoops: uint32(value.execution.configured_event_loops),
			ActiveEventLoops:     uint32(value.execution.active_event_loops),
			DrainingEventLoops:   uint32(value.execution.draining_event_loops),
			FailedEventLoops:     uint32(value.execution.failed_event_loops),
			Observed:             value.execution.observed != 0,
		},
		RuntimeInspection: RuntimeInspection{
			ABIVersion:        uint32(value.abi_version),
			Features:          features,
			IOUringCompiled:   value.io_uring_compiled != 0,
			IOUringProbed:     value.io_uring_probed != 0,
			IOUringSupported:  value.io_uring_supported != 0,
			IOUringProbeError: int32(value.io_uring_probe_error),
		},
		RequestedIOBackend:            requested,
		EffectiveIOBackend:            effective,
		FallbackReason:                IOBackendFallbackReason(value.io_fallback_reason),
		ServiceStarted:                value.service_started != 0,
		RequestNotificationBatchSize:  uint32(value.request_notification_batch_size),
		TerminalNotificationBatchSize: uint32(value.terminal_notification_batch_size),
		EventLoopCPUMask:              uint64(value.event_loop_cpu_mask),
		EventLoopCPUMaskSupported:     value.event_loop_cpu_mask_supported != 0,
		EventLoopCPUMaskApplied:       value.event_loop_cpu_mask_applied != 0,
	}
}

func nativeRuntimeInfo(_ *C.coakka_http_go_api_t) (RuntimeInspection, error) {
	var value C.coakka_http_host_runtime_info_t
	C.coakka_http_host_runtime_info_init(&value)
	if err := resultError(C.coakka_http_host_runtime_inspect(&value)); err != nil {
		return RuntimeInspection{}, err
	}
	return RuntimeInspection{
		ABIVersion: uint32(value.abi_version), Features: Capabilities(value.features),
		IOUringCompiled:   value.io_uring_compiled != 0,
		IOUringProbed:     value.io_uring_probed != 0,
		IOUringSupported:  value.io_uring_supported != 0,
		IOUringProbeError: int32(value.io_uring_probe_error),
	}, nil
}

func (native *nativeService) runtimeInfo() (RuntimeInfo, error) {
	var value C.coakka_http_host_service_info_t
	C.coakka_http_go_service_info_init(native.api, &value)
	if err := resultError(C.coakka_http_go_service_info(native.api, native.service, &value)); err != nil {
		return RuntimeInfo{}, err
	}
	var limits C.coakka_http_host_limits_t
	if err := resultError(C.coakka_http_go_service_limits(native.api, native.service, &limits)); err != nil {
		return RuntimeInfo{}, err
	}
	info := projectRuntimeInfo(value, native.capabilities)
	info.Limits = projectEffectiveLimits(limits)
	return info, nil
}

func closeNativeAPI(api **C.coakka_http_go_api_t) {
	C.coakka_http_go_close(api)
}

func (native *nativeService) destroy() error {
	if native == nil {
		return nil
	}
	if native.service != nil {
		// A refusal preserves the service and API together. Closing only the
		// latter would make the still-live owner unreachable on the next retry.
		if err := resultError(C.coakka_http_go_service_destroy(native.api, &native.service)); err != nil {
			return err
		}
	}
	C.coakka_http_go_close(&native.api)
	return nil
}
