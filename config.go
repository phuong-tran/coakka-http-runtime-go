package coakkahttp

// TransportTimeouts groups Core-owned connection deadlines. Zero delegates
// that field to Core's default; it does not disable the deadline. All values
// are milliseconds, resolved at startup and enforced using Core's monotonic
// clock. Go does not implement a second transport timer.
type TransportTimeouts struct {
	// HeaderMillis bounds receipt of request headers.
	HeaderMillis uint64
	// BodyMillis bounds receipt of the request body.
	BodyMillis uint64
	// KeepAliveMillis bounds idle time between persistent HTTP requests.
	KeepAliveMillis uint64
	// IdleMillis bounds connection idle time independently of keep-alive.
	IdleMillis uint64
}

// InspectionOptions enables Core-owned, authenticated runtime inspection.
// The runtime identity values identify the target instance and its generation.
type InspectionOptions struct {
	// RuntimeInstanceID is the nonzero target instance identity.
	RuntimeInstanceID uint64
	// RuntimeGeneration is the nonzero target instance generation.
	RuntimeGeneration uint64
	// BearerFile is the path to a token file read by Core.
	BearerFile string
	// AllowPlaintextLoopback explicitly permits inspection on a numeric
	// loopback plaintext listener.
	AllowPlaintextLoopback bool
}

// Config is a complete copied declaration for one Runtime instance. Nil optional
// sections retain Runtime defaults; all slices are copied during Open.
type Config struct {
	// CPU is Core's dispatcher placement policy; zero is CPUAuto. It does not
	// change GOMAXPROCS, existing Go workers or process-wide CPU affinity.
	CPU CPUPolicy
	// Limits optionally overrides Runtime's conservative resource defaults.
	Limits *RuntimeLimits
	// Listeners declares Runtime-owned inbound sockets.
	Listeners []Listener
	// Routes declares inbound routing and body-delivery policy.
	Routes []Route
	// StaticMounts declares Runtime-owned static and SPA file trees.
	StaticMounts []StaticMount
	// FileAuthorities declares roots allowed for application file responses.
	FileAuthorities []FileAuthority
	// Compression optionally configures bounded response compression.
	Compression *Compression
	// IOBackend optionally selects a native I/O backend; zero retains the default.
	IOBackend IOBackend
	// Monitor optionally reserves monitoring resources.
	Monitor *MonitorOptions
	// Inspection optionally enables authenticated runtime inspection. Nil leaves it off.
	Inspection *InspectionOptions
	// OutboundTargets declares named target topology copied by Runtime.
	OutboundTargets []OutboundTarget
	// OutboundTrust declares generation-keyed CA material copied by Runtime.
	OutboundTrust []OutboundTrust
	// OutboundIdentities declares generation-keyed client identities copied by Runtime.
	OutboundIdentities []OutboundIdentity
}

// Listener declares one Runtime-owned listening endpoint and optional file-backed
// TLS identity. Certificate, key, and trust files remain under Runtime ownership.
type Listener struct {
	// Protocol selects HTTP/1.1, HTTP/2, or HTTP/3; zero uses Core's default.
	Protocol ListenerProtocol
	// BindAddress is the host or address copied by Runtime; empty uses Core's loopback default.
	BindAddress string
	// Port selects a fixed port; zero requests an available port.
	Port uint16
	// Security selects plaintext, TLS, or mutual TLS; zero uses Core's default.
	Security TransportSecurity
	// CredentialGeneration identifies the TLS material revision.
	CredentialGeneration uint64
	// CredentialID is a stable application credential label.
	CredentialID string
	// CertificateChainFile is the Runtime-opened certificate chain path.
	CertificateChainFile string
	// PrivateKeyFile is the Runtime-opened private key path.
	PrivateKeyFile string
	// TrustRootsFile is the Runtime-opened client trust path for mutual TLS.
	TrustRootsFile string
}

// BodyPolicy constrains accepted request content types and body size.
type BodyPolicy struct {
	// Enabled applies the policy when true.
	Enabled bool
	// AcceptAbsent permits requests without a content type.
	AcceptAbsent bool
	// AcceptOther permits content types outside the two form types.
	AcceptOther bool
	// AcceptFormURLEncoded permits application/x-www-form-urlencoded.
	AcceptFormURLEncoded bool
	// AcceptMultipartFormData permits multipart/form-data.
	AcceptMultipartFormData bool
	// MaxBodyBytes bounds one accepted request body.
	MaxBodyBytes uint64
}

// Route declares one immutable Runtime route and its application binding.
type Route struct {
	// ID is the stable route identifier.
	ID uint64
	// HandlerBindingID selects the application handler generation.
	HandlerBindingID uint64
	// Method is the normalized HTTP method copied by Runtime.
	Method string
	// Pattern is the encoded absolute route pattern copied by Runtime.
	Pattern string
	// BodyDelivery selects inline or streamed request data.
	BodyDelivery BodyDelivery
	// BodyPolicy constrains admission for request bodies.
	BodyPolicy BodyPolicy
	// WebSocket marks this route as an HTTP upgrade endpoint.
	WebSocket bool
	// APIMetadata optionally documents this route through Core's neutral contract.
	APIMetadata *RouteAPIMetadata
}

// RuntimeLimits contains construction-time finite resource ceilings. Zero-valued
// fields select the corresponding Runtime default.
type RuntimeLimits struct {
	// MaxRoutes bounds route declarations.
	MaxRoutes uint32
	// MaxTotalRouteSegments bounds segments across the complete route table.
	MaxTotalRouteSegments uint32
	// MaxRouteBytes bounds the encoded route declaration storage.
	MaxRouteBytes uint32
	// MaxTotalRouteMetadataBytes bounds aggregate route-documentation storage.
	MaxTotalRouteMetadataBytes uint64
	// MaxCapturesPerRoute bounds named captures in one route.
	MaxCapturesPerRoute uint32
	// ActivationHistoryCapacity bounds idempotent control-operation history.
	ActivationHistoryCapacity uint32
	// MaxConnections bounds retained connections.
	MaxConnections uint32
	// MaxActiveExchanges bounds concurrent inbound exchanges.
	MaxActiveExchanges uint32
	// RequestQueueCapacity bounds queued inbound work.
	RequestQueueCapacity uint32
	// CompletionQueueCapacity bounds queued application completions.
	CompletionQueueCapacity uint32
	// CompletionBatchSize bounds completions handled per runtime turn.
	CompletionBatchSize uint32
	// MaxHeaderCount bounds headers on one message.
	MaxHeaderCount uint32
	// MaxHeaderBytes bounds total header bytes on one message.
	MaxHeaderBytes uint32
	// RequestStreamChunkSlots bounds queued request chunks.
	RequestStreamChunkSlots uint32
	// RequestStreamMaxRetainedItems bounds retained request stream items.
	RequestStreamMaxRetainedItems uint32
	// ResponseStreamChunkSlots bounds queued response chunks.
	ResponseStreamChunkSlots uint32
	// ResponseStreamMaxRetainedItems bounds retained response stream items.
	ResponseStreamMaxRetainedItems uint32
	// RouteControlCapacity bounds pending route-control commands.
	RouteControlCapacity uint32
	// RouteControlHistoryCapacity bounds retained route-control outcomes.
	RouteControlHistoryCapacity uint32
	// RouteControlMaxCommandsPerTurn bounds control work per runtime turn.
	RouteControlMaxCommandsPerTurn uint32
	// TerminalQueueCapacity bounds terminal notification retention.
	TerminalQueueCapacity uint32
	// DeadlineTickMillis selects the bounded deadline scan interval.
	DeadlineTickMillis uint32
	// ConnectionTimeoutBatchSize bounds timeout work per runtime turn.
	ConnectionTimeoutBatchSize uint32
	// Notifications selects startup notification profiles; zero uses Core defaults.
	Notifications NotificationProfiles
	// EventLoopCPUMask optionally selects Linux logical CPUs for the server event loop; zero inherits OS placement.
	EventLoopCPUMask uint64
	// MaxRequestBodyBytes bounds one request body.
	MaxRequestBodyBytes uint64
	// MaxResponseBodyBytes bounds one response body.
	MaxResponseBodyBytes uint64
	// KeepAliveTimeoutMillis bounds idle keep-alive connections.
	KeepAliveTimeoutMillis uint64
	// HeaderTimeoutMillis bounds request header receipt.
	HeaderTimeoutMillis uint64
	// BodyTimeoutMillis bounds request body receipt.
	BodyTimeoutMillis uint64
	// AppHostTimeoutMillis bounds application response latency.
	AppHostTimeoutMillis uint64
	// IdleTimeoutMillis bounds protocol idle time.
	IdleTimeoutMillis uint64
	// DrainTimeoutMillis bounds graceful drain.
	DrainTimeoutMillis uint64
	// StopTimeoutMillis bounds forced stop.
	StopTimeoutMillis uint64
	// ResponseWriteTimeoutMillis bounds stream backpressure waits.
	ResponseWriteTimeoutMillis uint64
	// RequestStreamMaxChunkBytes bounds one request stream chunk.
	RequestStreamMaxChunkBytes uint64
	// RequestStreamMaxTrailerBytes bounds one request trailer block.
	RequestStreamMaxTrailerBytes uint64
	// RequestStreamMaxRetainedBytes bounds one request stream's retained bytes.
	RequestStreamMaxRetainedBytes uint64
	// RequestStreamMaxTotalRetainedBytes bounds retained request stream bytes.
	RequestStreamMaxTotalRetainedBytes uint64
	// ResponseStreamMaxChunkBytes bounds one response stream write.
	ResponseStreamMaxChunkBytes uint64
	// ResponseStreamMaxTrailerBytes bounds one response trailer block.
	ResponseStreamMaxTrailerBytes uint64
	// ResponseStreamMaxRetainedBytes bounds one response stream's retained bytes.
	ResponseStreamMaxRetainedBytes uint64
	// ResponseStreamMaxTotalRetainedBytes bounds retained response stream bytes.
	ResponseStreamMaxTotalRetainedBytes uint64
}

// StaticMount declares one Runtime-owned static tree with optional index, cache,
// and SPA fallback behavior.
type StaticMount struct {
	// URLPrefix is the encoded URL path prefix.
	URLPrefix string
	// RootPath is the filesystem root opened by Runtime.
	RootPath string
	// IndexFile is the optional directory index path.
	IndexFile string
	// CacheControl is the optional emitted cache-control value.
	CacheControl string
	// SPAFallbackFile is the optional fallback path for unmatched frontend routes.
	SPAFallbackFile string
	// IncludeDotfiles permits dot-prefixed paths when true.
	IncludeDotfiles bool
	// HasIndexFile distinguishes an empty index value from no index policy.
	HasIndexFile bool
	// HasCacheControl distinguishes an empty value from no cache header.
	HasCacheControl bool
	// HasSPAFallbackFile distinguishes an empty fallback from no SPA policy.
	HasSPAFallbackFile bool
}

// FileAuthority declares a Runtime-owned filesystem root available to application
// file responses.
type FileAuthority struct {
	// ID is the stable authority identifier.
	ID uint64
	// RootPath is the filesystem root opened by Runtime.
	RootPath string
	// MaxActiveFiles bounds simultaneously active files for this authority.
	MaxActiveFiles uint32
	// MaxFileBytes bounds one served file.
	MaxFileBytes uint64
}

// Compression configures bounded response compression.
type Compression struct {
	// Mode selects disabled or gzip behavior.
	Mode CompressionMode
	// MinimumBodyBytes is the smallest eligible body.
	MinimumBodyBytes uint64
	// MaximumBodyBytes is the largest eligible input body.
	MaximumBodyBytes uint64
	// MaximumEncodedBytes bounds encoded output.
	MaximumEncodedBytes uint64
	// WorkspaceBytes bounds native compression workspace.
	WorkspaceBytes uint64
	// GZIPLevel selects the gzip compression level.
	GZIPLevel int32
}

// MonitorOptions reserves bounded construction-time monitoring resources.
type MonitorOptions struct {
	// Collection selects disabled, aggregate, or event collection.
	Collection MonitorCollection
	// EventCapacity reserves retained event slots.
	EventCapacity uint32
	// MaxEventsPerRead bounds one read page.
	MaxEventsPerRead uint32
	// AggregateCategories selects aggregate categories.
	AggregateCategories MonitorCategory
	// EventCategories selects retained event categories.
	EventCategories MonitorCategory
	// SignalReserved reserves native notification signaling.
	SignalReserved bool
	// FixedLatencyBuckets reserves the fixed histogram shape.
	FixedLatencyBuckets bool
}
