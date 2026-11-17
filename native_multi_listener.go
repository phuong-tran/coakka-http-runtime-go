package coakkahttp

/*
#include "bridge.h"
*/
import "C"

import "net/netip"

func (native *nativeService) multiProtocolListenerInfo() (MultiProtocolListenerInfo, error) {
	var value C.coakka_http_host_multi_listener_info_v1_t
	if err := resultError(C.coakka_http_go_multi_listener_inspect(
		&native.inspectMulti, native.service, &value)); err != nil {
		return MultiProtocolListenerInfo{}, err
	}
	var address netip.Addr
	switch value.address_family {
	case 4:
		var bytes [4]byte
		for index := range bytes {
			bytes[index] = byte(value.address[index])
		}
		address = netip.AddrFrom4(bytes)
	case 6:
		var bytes [16]byte
		for index := range bytes {
			bytes[index] = byte(value.address[index])
		}
		address = netip.AddrFrom16(bytes)
	}
	return MultiProtocolListenerInfo{
		State:                 ListenerState(value.lifecycle),
		AcceptedProtocols:     ListenerProtocols(value.accepted_protocols),
		AdmittingProtocols:    ListenerProtocols(value.admitting_protocols),
		BoundTransports:       ListenerTransports(value.bound_transports),
		Security:              TransportSecurity(value.security),
		Address:               address,
		Port:                  uint16(value.port),
		StartFailurePhase:     ListenerStartPhase(value.start_failure_phase),
		StartFailureTransport: ListenerTransports(value.start_failure_transport),
	}, nil
}
