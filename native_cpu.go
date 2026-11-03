package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import "errors"

// nativeCPUScope never crosses its entering OS thread. Core owns selection,
// probes, verification and restoration; Go only projects the immutable result.
type nativeCPUScope struct {
	value *C.coakka_http_host_cpu_scope_t
}

func enterNativeCPU(policy CPUPolicy) (*nativeCPUScope, CPUInfo, error) {
	scope := &nativeCPUScope{}
	var issue C.coakka_http_host_issue_t
	C.coakka_http_host_issue_init(&issue)
	if err := issueError(C.coakka_http_host_cpu_scope_enter(
		C.coakka_http_host_cpu_policy_t(policy), &scope.value, &issue), &issue); err != nil {
		return nil, CPUInfo{}, err
	}
	var value C.coakka_http_host_cpu_info_t
	value.struct_size = C.uint32_t(C.sizeof_coakka_http_host_cpu_info_t)
	if err := resultError(C.coakka_http_host_cpu_scope_info(scope.value, &value)); err != nil {
		return nil, CPUInfo{}, errors.Join(err, scope.leave())
	}
	return scope, CPUInfo{
		RequestedPolicy:   CPUPolicy(value.requested_policy),
		Placement:         CPUPlacement(value.placement),
		AvailableCPUCount: uint32(value.available_cpu_count),
		SelectedCPUCount:  uint32(value.selected_cpu_count),
		SelectedCPUIDs:    [2]uint32{uint32(value.selected_cpu_ids[0]), uint32(value.selected_cpu_ids[1])},
	}, nil
}

func (scope *nativeCPUScope) leave() error {
	var issue C.coakka_http_host_issue_t
	C.coakka_http_host_issue_init(&issue)
	return issueError(C.coakka_http_host_cpu_scope_leave(&scope.value, &issue), &issue)
}
