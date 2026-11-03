package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import "errors"

// routes reserves Core's immutable accepted maximum, not a racy observed count.
// The second call supplies every entry and its identity under one Core lock.
// Only scalar Go memory crosses this synchronous call; native retains nothing.
func (native *nativeService) routes() (RouteSnapshot, error) {
	info, err := native.runtimeInfo()
	if err != nil {
		return RouteSnapshot{}, err
	}
	capacity := info.Limits.MaxRoutes
	if capacity == 0 || capacity > maximumProjectionItems {
		return RouteSnapshot{}, errors.New("Core route capacity exceeds the Go projection bound")
	}
	entries := make([]C.coakka_http_host_route_state_t, int(capacity))
	var identity C.coakka_http_host_route_identity_t
	C.coakka_http_go_route_identity_init(native.api, &identity)
	if err := resultError(C.coakka_http_go_route_snapshot(native.api, native.service,
		&entries[0], C.uint32_t(capacity), &identity)); err != nil {
		return RouteSnapshot{}, err
	}
	if uint32(identity.route_count) > capacity {
		return RouteSnapshot{}, errors.New("Core route snapshot exceeds supplied storage")
	}
	routes := make([]RouteState, int(identity.route_count))
	for index := range routes {
		entry := entries[index]
		routes[index] = RouteState{
			RouteID: uint64(entry.route_id), HandlerBindingID: uint64(entry.handler_binding_id),
			BindingRevision: uint64(entry.binding_revision),
		}
	}
	return RouteSnapshot{
		RouteGeneration:       uint64(identity.route_generation),
		MetadataGeneration:    uint64(identity.metadata_generation),
		BindingChangeSequence: uint64(identity.binding_change_sequence), Routes: routes,
	}, nil
}
