package coakkahttp

import "net/netip"

// MultiProtocolListener declares one TLS endpoint for HTTP/1.1, HTTP/2 and
// HTTP/3. The same address and port apply to TCP and UDP. Core owns credential
// loading, negotiation and admission. A zero port asks Core to choose a port.
type MultiProtocolListener struct {
	// BindAddress is the numeric address or host passed to Core. Empty uses Core's default.
	BindAddress string
	// Port identifies the requested TCP and UDP port. Zero asks Core to choose one.
	Port uint16
	// CredentialGeneration identifies the configured TLS material.
	CredentialGeneration uint64
	// CredentialID labels the TLS identity.
	CredentialID string
	// CertificateChainFile names the certificate chain file read by Core.
	CertificateChainFile string
	// PrivateKeyFile names the private key file read by Core.
	PrivateKeyFile string
}

func (listener MultiProtocolListener) coreListener() Listener {
	return Listener{
		BindAddress:          listener.BindAddress,
		Port:                 listener.Port,
		Security:             SecurityTLS,
		CredentialGeneration: listener.CredentialGeneration,
		CredentialID:         listener.CredentialID,
		CertificateChainFile: listener.CertificateChainFile,
		PrivateKeyFile:       listener.PrivateKeyFile,
	}
}

// OpenMultiProtocol opens a Runtime with the same-port buffered TLS profile.
// Supply the endpoint in listener and leave config.Listeners empty. As with
// Open, the caller owns Start, the event reader and shutdown.
func OpenMultiProtocol(config Config, listener MultiProtocolListener) (*Runtime, error) {
	if len(config.Listeners) != 0 {
		return nil, &Error{Code: ResultInvalidArgument, Detail: "configure only the multi-protocol listener"}
	}
	config.Listeners = []Listener{listener.coreListener()}
	return openRuntime(config, true)
}

// ListenerProtocols is the set of HTTP protocols reported by a listener.
type ListenerProtocols uint32

const (
	// ListenerHTTP1 identifies HTTP/1.1.
	ListenerHTTP1 ListenerProtocols = 1 << iota
	// ListenerHTTP2 identifies HTTP/2.
	ListenerHTTP2
	// ListenerHTTP3 identifies HTTP/3.
	ListenerHTTP3
)

// ListenerTransports is the set of transports reported by a listener.
type ListenerTransports uint32

const (
	// ListenerTCP identifies the TCP transport.
	ListenerTCP ListenerTransports = 1 << iota
	// ListenerUDP identifies the UDP transport.
	ListenerUDP
)

// ListenerState is the Core-owned state of a multi-protocol listener.
type ListenerState uint32

const (
	// ListenerCreated indicates that the listener has yet to start.
	ListenerCreated ListenerState = iota
	// ListenerRunning indicates that the listener is admitting traffic.
	ListenerRunning
	// ListenerDraining indicates that admission has stopped and existing work is draining.
	ListenerDraining
	// ListenerStopped indicates that the listener has released its transports.
	ListenerStopped
	// ListenerFailed indicates that Core could not start or run the endpoint.
	ListenerFailed
)

// ListenerStartPhase identifies the Core stage that refused startup.
type ListenerStartPhase uint32

const (
	// ListenerStartNone indicates that Core recorded no startup failure.
	ListenerStartNone ListenerStartPhase = iota
	// ListenerStartPreparation identifies startup preparation.
	ListenerStartPreparation
	// ListenerStartTCPBind identifies the TCP bind.
	ListenerStartTCPBind
	// ListenerStartUDPBind identifies the UDP bind.
	ListenerStartUDPBind
	// ListenerStartActivation identifies activation of the protocol owners.
	ListenerStartActivation
	// ListenerStartPublication identifies publication of the endpoint.
	ListenerStartPublication
)

// MultiProtocolListenerInfo is a snapshot of Core's effective listener state.
// Address is invalid until Core publishes a bound address. An unbound
// listener reports port zero and an empty BoundTransports mask.
type MultiProtocolListenerInfo struct {
	// State is Core's current listener state.
	State ListenerState
	// AcceptedProtocols contains the protocol set that Core accepted for this endpoint.
	AcceptedProtocols ListenerProtocols
	// AdmittingProtocols contains the protocols accepting new requests at observation time.
	AdmittingProtocols ListenerProtocols
	// BoundTransports contains the sockets currently held by Core.
	BoundTransports ListenerTransports
	// Security is the observed security mode. Zero means that no endpoint is bound.
	Security TransportSecurity
	// Address is Core's bound address.
	Address netip.Addr
	// Port is the actual numeric port shared by TCP and UDP.
	Port uint16
	// StartFailurePhase is the startup phase reported by Core after a failure.
	StartFailurePhase ListenerStartPhase
	// StartFailureTransport identifies the transport involved in a bind failure.
	StartFailureTransport ListenerTransports
}

// MultiProtocolListenerInfo observes the same-port endpoint. A Runtime opened
// with the standalone listener API returns a typed unsupported error.
func (runtime *Runtime) MultiProtocolListenerInfo() (MultiProtocolListenerInfo, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return MultiProtocolListenerInfo{}, err
	}
	defer done.release()
	return native.multiProtocolListenerInfo()
}

// MultiProtocolListenerInfo reports the Core-owned endpoint state of a Service.
func (service *Service) MultiProtocolListenerInfo() (MultiProtocolListenerInfo, error) {
	if service == nil {
		return MultiProtocolListenerInfo{}, errRuntimeClosed
	}
	return service.runtime.MultiProtocolListenerInfo()
}
