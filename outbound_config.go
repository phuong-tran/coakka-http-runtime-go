package coakkahttp

// OutboundEndpoint declares one Runtime-owned connection destination.
type OutboundEndpoint struct {
	// Weight participates in weighted round-robin selection.
	Weight uint32
	// NodeID is the stable topology node identifier.
	NodeID string
	// ConnectHost is the DNS name or address Runtime connects to.
	ConnectHost string
	// ConnectPort is the destination port.
	ConnectPort uint16
	// HTTPAuthority is the HTTP Host or :authority value.
	HTTPAuthority string
	// Security selects plaintext, TLS, or mutual TLS.
	Security TransportSecurity
	// Available optionally controls endpoint admission. Nil keeps Runtime's
	// available-by-default initializer; a pointer to false disables the endpoint.
	Available *bool
	// TLSPeerIdentity is the expected peer identity.
	TLSPeerIdentity string
	// TLSTrustGeneration selects an added trust bundle.
	TLSTrustGeneration uint64
	// TLSClientIdentityGeneration selects an added client identity.
	TLSClientIdentityGeneration uint64
}

// OutboundTarget declares a named, generation-scoped endpoint set.
type OutboundTarget struct {
	// Name is the logical target used by outbound requests.
	Name string
	// Generation versions the target topology.
	Generation uint64
	// Strategy selects single-owner or weighted round-robin routing.
	Strategy OutboundStrategy
	// Endpoints are copied into Runtime in declaration order.
	Endpoints []OutboundEndpoint
}

// OutboundTrust supplies one generation-keyed CA bundle copied by Runtime.
type OutboundTrust struct {
	// Generation is referenced by outbound endpoints.
	Generation uint64
	// CAPEM contains PEM-encoded trust roots.
	CAPEM []byte
}

// OutboundIdentity supplies one generation-keyed mutual-TLS identity.
type OutboundIdentity struct {
	// Generation is referenced by outbound endpoints.
	Generation uint64
	// CertificateChainPEM contains the PEM certificate chain.
	CertificateChainPEM []byte
	// PrivateKeyPEM contains the PEM private key.
	PrivateKeyPEM []byte
}
