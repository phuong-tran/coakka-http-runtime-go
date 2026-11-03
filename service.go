package coakkahttp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

const (
	maximumRoutes          = 1024
	maximumHandlerBindings = 16384
	maximumStaticMounts    = 16
	maximumFileAuthorities = 16
	maximumOutboundTargets = 32
	maximumOutboundTrusts  = 16
	maximumOutboundIDs     = 16
	servicePollInterval    = 100 * time.Millisecond
)

var serviceMethods = map[string]struct{}{
	"DELETE": {},
	"GET":    {},
	"PATCH":  {},
	"POST":   {},
	"PUT":    {},
}

// Limits declares optional Core resource/deadline overrides and the Go handler
// binding bound. Zero Core fields select Core defaults, resolved at Start.
// RuntimeInfo().Limits reports accepted values, including dependent defaults.
type Limits struct {
	// MaxConnections bounds accepted connections retained by Runtime.
	MaxConnections uint32
	// MaxActiveRequests bounds concurrent exchanges retained by Runtime.
	MaxActiveRequests uint32
	// RequestQueueCapacity bounds requests waiting for a Go worker.
	RequestQueueCapacity uint32
	// ResponseQueueCapacity bounds native response completions.
	ResponseQueueCapacity uint32
	// MaxRequestBodyBytes bounds one inline request body.
	MaxRequestBodyBytes uint64
	// MaxResponseBodyBytes bounds one buffered response body.
	MaxResponseBodyBytes uint64
	// MaxTotalRouteMetadataBytes bounds aggregate Core-owned documentation.
	MaxTotalRouteMetadataBytes uint64
	// RequestTimeoutMillis bounds application response latency in Runtime.
	RequestTimeoutMillis uint32
	// ShutdownTimeoutMillis overrides Core drain and stop budgets. The Go worker
	// quiescence window uses Core's accepted drain budget, not a separate default.
	ShutdownTimeoutMillis uint32
	// ResponseWriteTimeoutMillis bounds each wait for response stream credit.
	ResponseWriteTimeoutMillis uint32
	// RequestStreamMaxRetainedItems bounds copied body events waiting per streamed request.
	RequestStreamMaxRetainedItems uint32
	// MaxHandlerBindings bounds Go-owned prepared/active/retired handlers. Zero
	// selects the Go binding-table default; it is not a native resource setting.
	MaxHandlerBindings uint32
	// Notifications selects startup notification profiles; zero uses Core defaults.
	Notifications NotificationProfiles
	// TransportTimeouts overrides grouped connection deadlines; zero fields
	// retain Core defaults, independently of the handler/write deadlines above.
	TransportTimeouts TransportTimeouts
	// EventLoopCPUMask selects Linux logical CPUs for the native server loop; zero inherits OS placement.
	EventLoopCPUMask uint64
}

// DefaultLimits returns a declaration that delegates every Core setting to
// Core. Its zero values mean unspecified, not unlimited. Only the Go-owned
// handler binding table has a connector default; inspect RuntimeInfo().Limits
// after Start for actual native values.
func DefaultLimits() Limits {
	return Limits{MaxHandlerBindings: 1_088}
}

// Validate checks only Go projection/allocation bounds. Core validates native
// limits and their relationships at Start, and owns all default resolution.
func (limits Limits) Validate() error {
	if limits.MaxRequestBodyBytes > maximumBridgeBytes ||
		limits.MaxResponseBodyBytes > maximumBridgeBytes ||
		limits.MaxHandlerBindings > maximumHandlerBindings {
		return errors.New("CoAkka HTTP service limits are invalid")
	}
	return nil
}

// validateEffectiveServiceLimits protects Go make/duration conversions. Native
// range relationships and timeout policy have already been checked by Core.
func validateEffectiveServiceLimits(limits RuntimeLimits) error {
	maximumInt := uint64(^uint(0) >> 1)
	for _, count := range []uint32{limits.MaxActiveExchanges, limits.RequestQueueCapacity, limits.RequestStreamMaxRetainedItems} {
		if count == 0 || uint64(count) > maximumInt {
			return errors.New("Core capacity cannot be represented by the Go service")
		}
	}
	const maximumWaitMillis = uint64((1<<63 - 1) / time.Millisecond)
	if limits.DrainTimeoutMillis == 0 || limits.DrainTimeoutMillis > maximumWaitMillis ||
		limits.ResponseWriteTimeoutMillis == 0 || limits.ResponseWriteTimeoutMillis > maximumWaitMillis ||
		limits.MaxRequestBodyBytes > maximumBridgeBytes || limits.MaxResponseBodyBytes > maximumBridgeBytes {
		return errors.New("Core limits exceed the Go projection bound")
	}
	return nil
}

type routeDeclaration struct {
	route     Route
	handler   Handler
	webSocket bool
}

// Builder collects one buffered inline-body Service declaration. Configure it
// from one goroutine and call Start once. Advanced and streaming applications
// should construct Config directly and use Open.
type Builder struct {
	cpu              CPUPolicy
	listener         Listener
	workers          uint32
	limits           Limits
	requestedBackend IOBackend
	monitor          *MonitorOptions
	inspection       *InspectionOptions
	compression      *Compression
	staticMounts     []StaticMount
	fileAuthorities  []FileAuthority
	outboundTargets  []OutboundTarget
	outboundTrusts   []OutboundTrust
	outboundIDs      []OutboundIdentity
	routes           []routeDeclaration
	started          bool
	deferred         error
}

// StaticMount adds one Runtime-owned static tree with optional index and SPA policy.
func (builder *Builder) StaticMount(mount StaticMount) *Builder {
	if builder.mutable() {
		if len(builder.staticMounts) >= maximumStaticMounts {
			builder.deferred = errors.New("static mount count exceeds the Runtime bound")
			return builder
		}
		builder.staticMounts = append(builder.staticMounts, mount)
	}
	return builder
}

// FileAuthority adds one bounded root for application-selected file responses.
func (builder *Builder) FileAuthority(authority FileAuthority) *Builder {
	if builder.mutable() {
		if len(builder.fileAuthorities) >= maximumFileAuthorities {
			builder.deferred = errors.New("file authority count exceeds the Runtime bound")
			return builder
		}
		builder.fileAuthorities = append(builder.fileAuthorities, authority)
	}
	return builder
}

// OutboundTarget adds one immutable logical target generation copied by Runtime.
func (builder *Builder) OutboundTarget(target OutboundTarget) *Builder {
	if builder.mutable() {
		if len(builder.outboundTargets) >= maximumOutboundTargets {
			builder.deferred = errors.New("outbound target count exceeds the Runtime bound")
			return builder
		}
		copy := target
		copy.Endpoints = append([]OutboundEndpoint(nil), target.Endpoints...)
		builder.outboundTargets = append(builder.outboundTargets, copy)
	}
	return builder
}

// OutboundTrust adds one generation-keyed CA bundle copied by Runtime.
func (builder *Builder) OutboundTrust(trust OutboundTrust) *Builder {
	if builder.mutable() {
		if len(builder.outboundTrusts) >= maximumOutboundTrusts {
			builder.deferred = errors.New("outbound trust count exceeds the Runtime bound")
			return builder
		}
		copy := trust
		copy.CAPEM = append([]byte(nil), trust.CAPEM...)
		builder.outboundTrusts = append(builder.outboundTrusts, copy)
	}
	return builder
}

// OutboundIdentity adds one generation-keyed mutual-TLS identity copied by Runtime.
func (builder *Builder) OutboundIdentity(identity OutboundIdentity) *Builder {
	if builder.mutable() {
		if len(builder.outboundIDs) >= maximumOutboundIDs {
			builder.deferred = errors.New("outbound identity count exceeds the Runtime bound")
			return builder
		}
		copy := identity
		copy.CertificateChainPEM = append([]byte(nil), identity.CertificateChainPEM...)
		copy.PrivateKeyPEM = append([]byte(nil), identity.PrivateKeyPEM...)
		builder.outboundIDs = append(builder.outboundIDs, copy)
	}
	return builder
}

// NewBuilder delegates native defaults to Core and uses one Go handler worker.
func NewBuilder() *Builder {
	return &Builder{
		workers: 1,
		limits:  DefaultLimits(),
	}
}

// Listen selects the bind host and port. Port zero requests an available port.
func (builder *Builder) Listen(host string, port uint16) *Builder {
	if !builder.mutable() {
		return builder
	}
	if strings.TrimSpace(host) == "" || strings.ContainsRune(host, '\x00') {
		builder.deferred = errors.New("listener host is invalid")
		return builder
	}
	builder.listener.BindAddress, builder.listener.Port = host, port
	return builder
}

// Listener replaces the complete single-listener protocol and TLS declaration.
func (builder *Builder) Listener(listener Listener) *Builder {
	if !builder.mutable() {
		return builder
	}
	builder.listener = listener
	return builder
}

// Concurrency selects the fixed number of Go handler workers.
func (builder *Builder) Concurrency(workers uint32) *Builder {
	if !builder.mutable() {
		return builder
	}
	if workers == 0 || workers > 1024 {
		builder.deferred = errors.New("handler worker count is invalid")
		return builder
	}
	builder.workers = workers
	return builder
}

// Limits replaces every principal Service resource ceiling.
func (builder *Builder) Limits(limits Limits) *Builder {
	if !builder.mutable() {
		return builder
	}
	if err := limits.Validate(); err != nil {
		builder.deferred = err
		return builder
	}
	builder.limits = limits
	return builder
}

// CPU selects Core's startup placement for this service's dispatcher and its
// native children. CPUAuto is the default; worker count and GOMAXPROCS are not
// changed. Read RuntimeInfo().CPU for Core's actual selection/limitation.
func (builder *Builder) CPU(policy CPUPolicy) *Builder {
	if builder.mutable() {
		builder.cpu = policy
	}
	return builder
}

// IOBackend selects a native I/O preference. The zero value and
// IOPlatformDefault keep platform I/O; IOUring opts in while native startup
// retains ownership of eligibility checks and fallback.
func (builder *Builder) IOBackend(backend IOBackend) *Builder {
	if !builder.mutable() {
		return builder
	}
	if backend != 0 && backend != IOPlatformDefault && backend != IOUring {
		builder.deferred = errors.New("I/O backend is invalid")
		return builder
	}
	builder.requestedBackend = backend
	return builder
}

// Monitor reserves bounded Runtime monitoring resources for this Service.
func (builder *Builder) Monitor(options MonitorOptions) *Builder {
	if !builder.mutable() {
		return builder
	}
	copy := options
	builder.monitor = &copy
	return builder
}

// Inspection copies the startup intent for authenticated runtime inspection.
// Core owns admission and verification of the bearer credential.
func (builder *Builder) Inspection(options InspectionOptions) *Builder {
	if builder.mutable() {
		builder.inspection = &options
	}
	return builder
}

// Compression copies optional response-compression intent for startup. Nil
// removes an earlier override and retains Core defaults. Core validates numeric
// bounds, availability and negotiation; Go does not compress handler responses.
// Configure the builder from one goroutine before Start, like other options.
func (builder *Builder) Compression(options *Compression) *Builder {
	if builder.mutable() {
		builder.compression = nil
		if options != nil {
			copy := *options
			builder.compression = &copy
		}
	}
	return builder
}

// Document snapshots metadata onto the most recently registered route. Core
// remains the authority for path, schema, response, and security semantics.
func (builder *Builder) Document(metadata RouteAPIMetadata) *Builder {
	if !builder.mutable() {
		return builder
	}
	if len(builder.routes) == 0 {
		builder.deferred = errors.New("route documentation requires a preceding route")
		return builder
	}
	index := len(builder.routes) - 1
	copy, err := cloneRouteAPIMetadata(&metadata)
	if err != nil {
		builder.deferred = err
		return builder
	}
	builder.routes[index].route.APIMetadata = copy
	return builder
}

// Handle registers a buffered inline-body handler for one method and pattern.
func (builder *Builder) Handle(method, pattern string, handler Handler) *Builder {
	return builder.handle(method, pattern, BodyInline, handler)
}

// HandleStream registers a handler whose request body is read incrementally.
func (builder *Builder) HandleStream(method, pattern string, handler Handler) *Builder {
	return builder.handle(method, pattern, BodyStream, handler)
}

// HandleWebSocket registers a route whose handler must select a WebSocket
// response. The declaration reserves the Runtime WebSocket lane before bind.
func (builder *Builder) HandleWebSocket(method, pattern string, handler Handler) *Builder {
	before := len(builder.routes)
	builder.handle(method, pattern, BodyInline, handler)
	if len(builder.routes) == before+1 {
		builder.routes[before].webSocket = true
	}
	return builder
}

// WebSocket registers one GET WebSocket upgrade route.
func (builder *Builder) WebSocket(pattern string, handler Handler) *Builder {
	return builder.HandleWebSocket("GET", pattern, handler)
}

func (builder *Builder) handle(method, pattern string, delivery BodyDelivery, handler Handler) *Builder {
	if !builder.mutable() {
		return builder
	}
	normalized := strings.ToUpper(strings.TrimSpace(method))
	_, supported := serviceMethods[normalized]
	if !supported || !strings.HasPrefix(pattern, "/") || handler == nil {
		builder.deferred = errors.New("route declaration is invalid")
		return builder
	}
	if len(builder.routes) >= maximumRoutes {
		builder.deferred = errors.New("route count exceeds the Service bound")
		return builder
	}
	id := uint64(len(builder.routes) + 1)
	builder.routes = append(builder.routes, routeDeclaration{
		route: Route{
			ID:               id,
			HandlerBindingID: id,
			Method:           normalized,
			Pattern:          pattern,
			BodyDelivery:     delivery,
		},
		handler: handler,
	})
	return builder
}

// Get registers a GET handler.
func (builder *Builder) Get(pattern string, handler Handler) *Builder {
	return builder.Handle("GET", pattern, handler)
}

// Post registers a POST handler.
func (builder *Builder) Post(pattern string, handler Handler) *Builder {
	return builder.Handle("POST", pattern, handler)
}

// Put registers a PUT handler.
func (builder *Builder) Put(pattern string, handler Handler) *Builder {
	return builder.Handle("PUT", pattern, handler)
}

// Patch registers a PATCH handler.
func (builder *Builder) Patch(pattern string, handler Handler) *Builder {
	return builder.Handle("PATCH", pattern, handler)
}

// Delete registers a DELETE handler.
func (builder *Builder) Delete(pattern string, handler Handler) *Builder {
	return builder.Handle("DELETE", pattern, handler)
}

// GetStream registers a streamed GET request body handler.
func (builder *Builder) GetStream(pattern string, handler Handler) *Builder {
	return builder.HandleStream("GET", pattern, handler)
}

// PostStream registers a streamed POST request body handler.
func (builder *Builder) PostStream(pattern string, handler Handler) *Builder {
	return builder.HandleStream("POST", pattern, handler)
}

// PutStream registers a streamed PUT request body handler.
func (builder *Builder) PutStream(pattern string, handler Handler) *Builder {
	return builder.HandleStream("PUT", pattern, handler)
}

// PatchStream registers a streamed PATCH request body handler.
func (builder *Builder) PatchStream(pattern string, handler Handler) *Builder {
	return builder.HandleStream("PATCH", pattern, handler)
}

// DeleteStream registers a streamed DELETE request body handler.
func (builder *Builder) DeleteStream(pattern string, handler Handler) *Builder {
	return builder.HandleStream("DELETE", pattern, handler)
}

func (builder *Builder) mutable() bool {
	if builder == nil {
		return false
	}
	if builder.started {
		builder.deferred = errors.New("builder can only be started once")
		return false
	}
	return true
}

// Start creates Runtime, starts listeners, and launches one bounded event pump
// plus the configured fixed worker set. If failed startup also cannot destroy
// Core, StartError.RetainedRuntime preserves that owner for a Close retry.
func (builder *Builder) Start() (*Service, error) {
	if builder == nil {
		return nil, errors.New("builder is nil")
	}
	if builder.started {
		return nil, errors.New("builder can only be started once")
	}
	builder.started = true
	if builder.deferred != nil {
		return nil, builder.deferred
	}
	if len(builder.routes) == 0 && len(builder.staticMounts) == 0 && len(builder.outboundTargets) == 0 {
		return nil, errors.New("at least one route, static mount, or outbound target is required")
	}
	if len(builder.outboundTargets) == 0 &&
		(len(builder.outboundTrusts) != 0 || len(builder.outboundIDs) != 0) {
		return nil, errors.New("outbound credentials require a logical target")
	}
	if err := builder.limits.Validate(); err != nil {
		return nil, err
	}
	if builder.limits.MaxHandlerBindings == 0 {
		builder.limits.MaxHandlerBindings = DefaultLimits().MaxHandlerBindings
	}
	if uint32(len(builder.routes)) > builder.limits.MaxHandlerBindings {
		return nil, errors.New("route count exceeds the handler binding capacity")
	}
	requiredCapabilities := Capabilities(CapabilityInbound | CapabilityRouteRebind |
		CapabilityRouteManifestPublication | CapabilityHealth |
		CapabilityResponseStream | CapabilityResponseTrailers | CapabilityResponseWriteTimeout |
		CapabilityServerSentEvents)
	usesWebSocket := false
	for _, declaration := range builder.routes {
		if declaration.route.BodyDelivery == BodyStream {
			requiredCapabilities |= CapabilityRequestStream
		}
		usesWebSocket = usesWebSocket || declaration.webSocket
	}
	if usesWebSocket {
		requiredCapabilities |= CapabilityWebSocket
	}
	if builder.listener.Protocol == ProtocolHTTP2 {
		requiredCapabilities |= CapabilityHTTP2
	} else if builder.listener.Protocol == ProtocolHTTP3 {
		requiredCapabilities |= CapabilityHTTP3
	}
	if builder.listener.Security == SecurityTLS {
		requiredCapabilities |= CapabilityTLS
	} else if builder.listener.Security == SecurityMutualTLS {
		requiredCapabilities |= CapabilityMutualTLS
	}
	if builder.monitor != nil {
		requiredCapabilities |= CapabilityMonitor
	}
	if builder.compression != nil && builder.compression.Mode == CompressionGZIP {
		requiredCapabilities |= CapabilityGZIP
	}
	if len(builder.staticMounts) != 0 {
		requiredCapabilities |= CapabilityStaticFiles | CapabilityStaticFrontend
	}
	if len(builder.fileAuthorities) != 0 {
		requiredCapabilities |= CapabilityApplicationFiles
	}
	if len(builder.outboundTargets) != 0 {
		requiredCapabilities |= CapabilityOutbound | CapabilityOutboundLogicalTarget
	}
	plan, err := PlanService(requiredCapabilities, CapabilityAll)
	if err != nil {
		return nil, err
	}
	if !plan.Supported {
		return nil, fmt.Errorf(
			"connector cannot provide the declared service: missing=%#x",
			uint64(plan.MissingCapabilities),
		)
	}
	limits := builder.limits
	routes := make([]Route, len(builder.routes))
	handlers := make(map[uint64]Handler, len(builder.routes))
	for index, declaration := range builder.routes {
		route := declaration.route
		route.WebSocket = declaration.webSocket
		routes[index] = route
		handlers[route.HandlerBindingID] = declaration.handler
	}
	routes, err = cloneRoutes(routes)
	if err != nil {
		return nil, err
	}
	runtime, err := Open(Config{
		CPU:       builder.cpu,
		IOBackend: builder.requestedBackend,
		Limits: &RuntimeLimits{
			MaxConnections:                limits.MaxConnections,
			MaxActiveExchanges:            limits.MaxActiveRequests,
			RequestQueueCapacity:          limits.RequestQueueCapacity,
			CompletionQueueCapacity:       limits.ResponseQueueCapacity,
			RequestStreamMaxRetainedItems: limits.RequestStreamMaxRetainedItems,
			MaxRequestBodyBytes:           limits.MaxRequestBodyBytes,
			MaxResponseBodyBytes:          limits.MaxResponseBodyBytes,
			MaxTotalRouteMetadataBytes:    limits.MaxTotalRouteMetadataBytes,
			AppHostTimeoutMillis:          uint64(limits.RequestTimeoutMillis),
			DrainTimeoutMillis:            uint64(limits.ShutdownTimeoutMillis),
			StopTimeoutMillis:             uint64(limits.ShutdownTimeoutMillis),
			ResponseWriteTimeoutMillis:    uint64(limits.ResponseWriteTimeoutMillis),
			Notifications:                 limits.Notifications,
			HeaderTimeoutMillis:           limits.TransportTimeouts.HeaderMillis,
			BodyTimeoutMillis:             limits.TransportTimeouts.BodyMillis,
			KeepAliveTimeoutMillis:        limits.TransportTimeouts.KeepAliveMillis,
			IdleTimeoutMillis:             limits.TransportTimeouts.IdleMillis,
			EventLoopCPUMask:              limits.EventLoopCPUMask,
		},
		Listeners:       []Listener{builder.listener},
		Routes:          routes,
		StaticMounts:    append([]StaticMount(nil), builder.staticMounts...),
		FileAuthorities: append([]FileAuthority(nil), builder.fileAuthorities...),
		Monitor:         builder.monitor,
		Inspection:      builder.inspection,
		Compression:     builder.compression,
		OutboundTargets: append([]OutboundTarget(nil), builder.outboundTargets...),
		OutboundTrust:   append([]OutboundTrust(nil), builder.outboundTrusts...),
		OutboundIdentities: append(
			[]OutboundIdentity(nil), builder.outboundIDs...,
		),
	})
	if err != nil {
		return nil, err
	}
	// Read accepted native bounds before creating Go queues or starting a
	// listener. A read/projection failure follows the same retained cleanup law
	// as a refused native start; no partially started Service is returned.
	info, err := runtime.RuntimeInfo()
	if err == nil {
		err = validateEffectiveServiceLimits(info.Limits)
	}
	if err == nil {
		err = runtime.Start()
	}
	if err != nil {
		if cleanupErr := runtime.Close(); cleanupErr != nil {
			failure := &StartError{Err: errors.Join(err, cleanupErr)}
			if runtime.retained() {
				failure.RetainedRuntime = runtime
			}
			return nil, failure
		}
		return nil, err
	}
	service := &Service{
		runtime:               runtime,
		plan:                  plan,
		routes:                routes,
		handlers:              handlers,
		maxHandlerBindings:    limits.MaxHandlerBindings,
		jobs:                  make(chan *Request, info.Limits.RequestQueueCapacity),
		pending:               make(map[ExchangeID]struct{}, info.Limits.MaxActiveExchanges),
		requestStreams:        make(map[ExchangeID]*requestBodyStream, info.Limits.MaxActiveExchanges),
		requestStreamCapacity: info.Limits.RequestStreamMaxRetainedItems,
		responseStreams:       make(map[ExchangeID]*responseStreamState, info.Limits.MaxActiveExchanges),
		responseWriteTimeout:  time.Duration(info.Limits.ResponseWriteTimeoutMillis) * time.Millisecond,
		shutdownTimeout:       time.Duration(info.Limits.DrainTimeoutMillis) * time.Millisecond,
		pumpDone:              make(chan struct{}),
		remainingWorkers:      builder.workers,
		workersDone:           make(chan struct{}),
	}
	if usesWebSocket {
		service.webSockets = newWebSocketDispatcher(service, info.Limits.MaxActiveExchanges)
		service.webSocketReaderDone = make(chan struct{})
	}
	for worker := uint32(0); worker < builder.workers; worker++ {
		go service.runWorker()
	}
	service.runtime.dispatcher.runPump(service.runPump)
	if service.webSockets != nil {
		go service.runWebSocketReader()
	}
	return service, nil
}

type diagnosticQueue struct {
	mu     sync.Mutex
	values []error
}

func (queue *diagnosticQueue) add(err error) {
	if err == nil {
		return
	}
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if len(queue.values) == 64 {
		copy(queue.values, queue.values[1:])
		queue.values = queue.values[:63]
	}
	queue.values = append(queue.values, err)
}

func (queue *diagnosticQueue) take() error {
	queue.mu.Lock()
	defer queue.mu.Unlock()
	if len(queue.values) == 0 {
		return nil
	}
	err := queue.values[0]
	copy(queue.values, queue.values[1:])
	queue.values = queue.values[:len(queue.values)-1]
	return err
}

// Service is a buffered Handler facade over the callback-free Runtime API.
// Close waits for admitted handlers; handlers must therefore return and must
// not call Close on their own Service.
type Service struct {
	closeMu               sync.Mutex
	closed                bool
	shutdownStarted       bool
	drainStarted          bool // closeMu: only a successful Core begin-drain acknowledgement.
	shutdownErr           error
	shutdownTimeout       time.Duration
	jobsClosed            bool
	stateMu               sync.Mutex
	closing               bool
	drainObserved         bool // stateMu: Core terminal observation, not local queue state.
	drainErr              error
	pending               map[ExchangeID]struct{}
	requestStreams        map[ExchangeID]*requestBodyStream
	requestStreamCapacity uint32
	responseStreams       map[ExchangeID]*responseStreamState
	responseWriteTimeout  time.Duration
	runtime               *Runtime
	plan                  ServicePlan
	routes                []Route
	routesMu              sync.RWMutex
	handlers              map[uint64]Handler
	handlersMu            sync.RWMutex
	maxHandlerBindings    uint32
	jobs                  chan *Request
	pumpDone              chan struct{}
	workersMu             sync.Mutex
	remainingWorkers      uint32
	workersDone           chan struct{}
	errors                diagnosticQueue
	webSockets            *webSocketDispatcher
	webSocketReaderDone   chan struct{}
}

// Plan returns Runtime's immutable whole-service capability decision made
// before listener publication.
func (service *Service) Plan() ServicePlan {
	if service == nil {
		return ServicePlan{}
	}
	return service.plan
}

// RuntimeInfo returns capability and effective backend truth copied from Runtime.
func (service *Service) RuntimeInfo() (RuntimeInfo, error) {
	if service == nil || service.runtime == nil {
		return RuntimeInfo{}, errors.New("service is closed")
	}
	return service.runtime.RuntimeInfo()
}

// Health returns a non-blocking copied Runtime health snapshot.
func (service *Service) Health() (Health, error) {
	if service == nil || service.runtime == nil {
		return Health{}, errors.New("service is closed")
	}
	return service.runtime.Health()
}

// ProbeLiveness waits for fresh Runtime progress evidence within timeout.
func (service *Service) ProbeLiveness(timeout time.Duration) (Health, error) {
	if service == nil || service.runtime == nil {
		return Health{}, errors.New("service is closed")
	}
	return service.runtime.ProbeLiveness(timeout)
}

// SubmitOutbound starts one buffered call to a startup-declared logical target.
func (service *Service) SubmitOutbound(request ClientRequest) (OutboundID, error) {
	if service == nil || service.runtime == nil {
		return OutboundID{}, errors.New("service is closed")
	}
	return service.runtime.SubmitOutbound(request)
}

// CancelOutbound requests cancellation of an admitted outbound call.
func (service *Service) CancelOutbound(call OutboundID) error {
	if service == nil || service.runtime == nil {
		return errors.New("service is closed")
	}
	return service.runtime.CancelOutbound(call)
}

// TakeOutbound waits for and copies one terminal, or returns nil on timeout.
// Exactly one goroutine may call TakeOutbound for a Service at a time.
func (service *Service) TakeOutbound(timeout time.Duration) (*ClientTerminal, error) {
	if service == nil || service.runtime == nil {
		return nil, errors.New("service is closed")
	}
	return service.runtime.TakeOutbound(timeout)
}

// MonitorConfig returns the copied effective monitoring configuration.
func (service *Service) MonitorConfig() (MonitorConfig, error) {
	if service == nil || service.runtime == nil {
		return MonitorConfig{}, errors.New("service is closed")
	}
	return service.runtime.MonitorConfig()
}

// MonitorSnapshot returns copied bounded aggregates and co-observed health.
func (service *Service) MonitorSnapshot() (MonitorSnapshot, error) {
	if service == nil || service.runtime == nil {
		return MonitorSnapshot{}, errors.New("service is closed")
	}
	return service.runtime.MonitorSnapshot()
}

// ApplyMonitorPolicy performs an optimistic live monitor policy update.
func (service *Service) ApplyMonitorPolicy(expectedGeneration uint64, policy MonitorPolicy) (MonitorApplyOutcome, error) {
	if service == nil || service.runtime == nil {
		return MonitorApplyOutcome{}, errors.New("service is closed")
	}
	return service.runtime.ApplyMonitorPolicy(expectedGeneration, policy)
}

// ReadMonitorEvents reads one bounded cursor page of copied monitor events.
func (service *Service) ReadMonitorEvents(afterSequence uint64, maximumEvents uint32) (MonitorEventPage, error) {
	if service == nil || service.runtime == nil {
		return MonitorEventPage{}, errors.New("service is closed")
	}
	return service.runtime.ReadMonitorEvents(afterSequence, maximumEvents)
}

// WaitMonitor waits until newer monitoring truth may be available.
func (service *Service) WaitMonitor(timeout time.Duration) error {
	if service == nil || service.runtime == nil {
		return errors.New("service is closed")
	}
	return service.runtime.WaitMonitor(timeout)
}

// InterruptMonitor wakes the sole monitor waiter without fabricating data.
func (service *Service) InterruptMonitor() error {
	if service == nil || service.runtime == nil {
		return errors.New("service is closed")
	}
	return service.runtime.InterruptMonitor()
}

// PrepareHandler registers one new single-assignment binding identity. It is
// not dispatchable until RebindHandler publishes it through Runtime.
func (service *Service) PrepareHandler(bindingID uint64, handler Handler) error {
	if service == nil || bindingID == 0 || handler == nil {
		return errors.New("handler binding is invalid")
	}
	service.closeMu.Lock()
	defer service.closeMu.Unlock()
	if service.closed || service.shutdownStarted {
		return errors.New("service is closing or closed")
	}
	service.handlersMu.Lock()
	defer service.handlersMu.Unlock()
	if _, exists := service.handlers[bindingID]; exists {
		return errors.New("handler binding identity was already used")
	}
	if uint32(len(service.handlers)) >= service.maxHandlerBindings {
		return errors.New("handler binding capacity is exhausted")
	}
	service.handlers[bindingID] = handler
	return nil
}

// RebindHandler publishes one previously prepared binding under Runtime's route
// generation and binding revision checks. Ambiguous outcomes can be retried
// with the exact same request because prepared bindings remain registered.
func (service *Service) RebindHandler(request RebindRequest) (RebindOutcome, error) {
	if service == nil || service.runtime == nil {
		return RebindOutcome{}, errors.New("service is closed")
	}
	service.closeMu.Lock()
	defer service.closeMu.Unlock()
	if service.closed || service.shutdownStarted {
		return RebindOutcome{}, errors.New("service is closing or closed")
	}
	service.handlersMu.RLock()
	_, prepared := service.handlers[request.NewHandlerBindingID]
	service.handlersMu.RUnlock()
	if !prepared {
		return RebindOutcome{}, errors.New("handler binding was not prepared")
	}
	outcome, err := service.runtime.Rebind(request)
	if err != nil || outcome.Code != RebindApplied {
		return outcome, err
	}
	current, err := service.currentControlCut(outcome.Replayed, outcome.RouteGeneration, outcome.BindingChangeSequence)
	if err != nil || !current {
		return outcome, err
	}
	service.routesMu.Lock()
	// Cold bounded declaration maintenance, never request dispatch. Core owns
	// the actual route lookup and captured binding identity.
	for index := range service.routes {
		if service.routes[index].ID == outcome.RouteID {
			service.routes[index].HandlerBindingID = outcome.EffectiveHandlerBindingID
			break
		}
	}
	service.routesMu.Unlock()
	return outcome, nil
}

// currentControlCut separates historical replay from a first resolved retry.
// Replayed alone cannot distinguish them. Inspection refusal preserves the Go
// declarations and returns the error; only Core can certify its current cut.
// The caller holds closeMu to serialize Service route-control operations.
func (service *Service) currentControlCut(replayed bool, generation, sequence uint64) (bool, error) {
	if !replayed {
		return true, nil
	}
	health, err := service.runtime.Health()
	if err != nil {
		return false, err
	}
	return health.RouteGeneration == generation && health.ChangeSequence == sequence, nil
}

// currentPublicationCut includes metadata's independent generation. A route
// replay cannot adopt declarations from a structurally equal but differently
// documented Core cut.
func (service *Service) currentPublicationCut(replayed bool, routeGeneration,
	metadataGeneration, sequence uint64) (bool, error) {
	if !replayed {
		return true, nil
	}
	health, err := service.runtime.Health()
	if err != nil {
		return false, err
	}
	return health.RouteGeneration == routeGeneration &&
		health.MetadataGeneration == metadataGeneration &&
		health.ChangeSequence == sequence, nil
}

// PublishRoutes atomically replaces the complete structural route generation.
// Every referenced binding must have been registered through PrepareHandler.
// Delivery-ambiguous outcomes can be resolved by retrying the exact request.
func (service *Service) PublishRoutes(request RoutePublication) (RoutePublicationOutcome, error) {
	if service == nil || service.runtime == nil {
		return RoutePublicationOutcome{}, errors.New("service is closed")
	}
	if len(request.Routes) > maximumHandlerBindings {
		return RoutePublicationOutcome{}, errors.New("route generation exceeds the service bound")
	}
	routes, err := cloneRoutes(request.Routes)
	if err != nil {
		return RoutePublicationOutcome{}, err
	}
	service.closeMu.Lock()
	defer service.closeMu.Unlock()
	if service.closed || service.shutdownStarted {
		return RoutePublicationOutcome{}, errors.New("service is closing or closed")
	}
	service.handlersMu.RLock()
	for _, route := range routes {
		if route.HandlerBindingID == 0 || service.handlers[route.HandlerBindingID] == nil {
			service.handlersMu.RUnlock()
			return RoutePublicationOutcome{}, errors.New("route references an unprepared handler binding")
		}
	}
	service.handlersMu.RUnlock()
	request.Routes = routes
	outcome, err := service.runtime.PublishRoutes(request)
	if err != nil {
		return RoutePublicationOutcome{}, err
	}
	if outcome.Code == RoutePublicationApplied {
		current, inspectErr := service.currentPublicationCut(outcome.Replayed,
			outcome.EffectiveRouteGeneration, outcome.EffectiveMetadataGeneration,
			outcome.EffectiveBindingChangeSequence)
		if inspectErr != nil || !current {
			return outcome, inspectErr
		}
		service.routesMu.Lock()
		service.routes = routes
		service.routesMu.Unlock()
	}
	return outcome, nil
}

// Port returns the selected port of the Service listener.
func (service *Service) Port() (uint16, error) {
	if service == nil {
		return 0, errors.New("service is closed")
	}
	return service.runtime.BoundPort()
}

// PollError removes and returns the oldest bounded pump or handler diagnostic.
func (service *Service) PollError() error {
	if service == nil {
		return nil
	}
	return service.errors.take()
}

func (service *Service) runPump() {
	defer close(service.pumpDone)
	for {
		event, err := service.runtime.takeDispatcherEvent(servicePollInterval)
		if err != nil {
			var runtimeErr *Error
			if errors.As(err, &runtimeErr) && (runtimeErr.Code == ResultTimeout || runtimeErr.Code == ResultClosed) {
				if service.pumpMayStop() {
					return
				}
				continue
			}
			if errors.Is(err, errRuntimeClosed) {
				return
			}
			service.errors.add(err)
			continue
		}
		if event.Kind == EventResponseWritable {
			service.signalResponseStreamWritable(event.Exchange)
			continue
		}
		if event.Kind == EventRequestCancelled {
			service.finishRequestStream(event.Exchange, errRequestBodyCancelled)
			service.signalResponseStreamTerminal(event.Exchange, &coreStreamEnd{kind: event.Kind})
			continue
		}
		if event.Kind == EventTerminal {
			service.observeTerminal(event.Exchange)
			continue
		}
		if event.Kind == EventRequestData || event.Kind == EventRequestTrailers || event.Kind == EventRequestEnd {
			service.routeRequestStreamEvent(event)
			continue
		}
		if event.Kind != EventRequest || event.Request == nil {
			continue
		}
		if service.trackRequest(event.Exchange) {
			if responseErr := service.runtime.Respond(event.Exchange, internalResponse()); responseErr != nil {
				if !retiredSubmission(responseErr) {
					service.errors.add(responseErr)
				}
			}
			continue
		}
		if event.Request.BodyDelivery == BodyStream {
			stream := newRequestBodyStream(
				event.Request,
				service.requestStreamCapacity,
				event.Sequence,
			)
			service.stateMu.Lock()
			service.requestStreams[event.Exchange] = stream
			service.stateMu.Unlock()
			event.Request.Body = stream
		} else {
			event.Request.Body = inlineRequestBody(event.Request.body)
		}
		select {
		case service.jobs <- event.Request:
		default:
			service.errors.add(errors.New("Go handler queue is full"))
			_ = service.runtime.Respond(event.Exchange, overloadedResponse())
		}
	}
}

// emptyRequestBody has no cursor or close state. Its value may be shared by
// unrelated requests without allocation or synchronization. Preserve the
// buffered reader's EOF, no-op Close and WriterTo behavior, including Read(nil).
type emptyRequestBody struct{}

// Read reports EOF for the empty body, including a zero-length destination.
func (emptyRequestBody) Read([]byte) (int, error) { return 0, io.EOF }

// Close is a no-op, matching buffered bodies without retaining close state.
func (emptyRequestBody) Close() error { return nil }

// WriteTo completes without calling the writer because no bytes are available.
func (emptyRequestBody) WriteTo(io.Writer) (int64, error) { return 0, nil }

func inlineRequestBody(body []byte) io.ReadCloser {
	if len(body) == 0 {
		return emptyRequestBody{}
	}
	// Nonempty bodies retain an independent cursor per request. This wraps the
	// already-owned bytes; it does not borrow native memory or recopy the body.
	return io.NopCloser(bytes.NewReader(body))
}

// trackRequest records every request event before it can be dispatched. Once
// shutdown begins, the pump still completes already-admitted Runtime exchanges
// with a bounded failure instead of abandoning their terminal observations.
func (service *Service) trackRequest(exchange ExchangeID) bool {
	service.stateMu.Lock()
	defer service.stateMu.Unlock()
	service.pending[exchange] = struct{}{}
	return service.closing
}

func (service *Service) observeTerminal(exchange ExchangeID) {
	service.stateMu.Lock()
	stream := service.responseStreams[exchange]
	requestStream := service.requestStreams[exchange]
	delete(service.responseStreams, exchange)
	delete(service.requestStreams, exchange)
	delete(service.pending, exchange)
	service.stateMu.Unlock()
	if stream != nil {
		stream.terminate(&coreStreamEnd{kind: EventTerminal})
	}
	if requestStream != nil {
		requestStream.finish(errRequestBodyTerminated)
	}
}

// retiredSubmission projects Core's explicit terminal/closed refusal for a
// Core-issued exchange. It never infers native state from a Go map or suppresses
// INVALID_ARGUMENT: malformed responses remain errors even after cancellation.
// A pulled Core end event is also authoritative; it is not a native call result.
func retiredSubmission(err error) bool {
	// Preserve every non-Core cause in errors.Join; one retired child must not
	// hide a producer failure. Bound inspection of application error wrappers,
	// including cyclic/custom chains, and retain the diagnostic on exhaustion.
	remaining := 64
	return onlyCoreRetirement(err, &remaining)
}

func onlyCoreRetirement(err error, remaining *int) bool {
	if err == nil || *remaining == 0 {
		return false
	}
	*remaining--
	switch value := err.(type) {
	case *Error:
		return value != nil && (value.Code == ResultStaleExchange || value.Code == ResultClosed)
	case *coreStreamEnd:
		return value != nil && (value.kind == EventTerminal || value.kind == EventRequestCancelled)
	case interface {
		// Unwrap exposes every joined cause; all must be Core retirement.
		Unwrap() []error
	}:
		causes := value.Unwrap()
		if len(causes) == 0 {
			return false
		}
		for _, cause := range causes {
			if !onlyCoreRetirement(cause, remaining) {
				return false
			}
		}
		return true
	case interface {
		// Unwrap preserves an ordinary Go error wrapper's causal chain.
		Unwrap() error
	}:
		return onlyCoreRetirement(value.Unwrap(), remaining)
	default:
		return false
	}
}

func (service *Service) beginShutdown() {
	service.stateMu.Lock()
	service.closing = true
	service.stateMu.Unlock()
}

// Core alone certifies native completion. The local pending ledger is only
// an additional Go observation barrier. After the certificate, take once more
// so a terminal published between the previous empty read and the certificate
// cannot be abandoned. No native query is added to the running request path.
func (service *Service) pumpMayStop() bool {
	service.stateMu.Lock()
	closing, observed, empty := service.closing, service.drainObserved, len(service.pending) == 0
	service.stateMu.Unlock()
	if !closing {
		return false
	}
	if observed {
		return empty
	}
	state, err := service.runtime.WaitDrained(0)
	switch state {
	case DrainNotStarted, DrainPending:
		// Close may have published its Go intent before Core acknowledged drain.
		return false
	case DrainComplete, DrainExpired, DrainAborted, DrainFailed:
		service.stateMu.Lock()
		service.drainObserved = true
		service.drainErr = err
		service.stateMu.Unlock()
	default:
		// Unsupported/unknown is a retained-owner error, never proof of safety.
		service.stateMu.Lock()
		service.drainErr = err
		service.stateMu.Unlock()
	}
	return false
}

func (service *Service) drainFailure() error {
	service.stateMu.Lock()
	defer service.stateMu.Unlock()
	return service.drainErr
}

func (service *Service) runWorker() {
	defer service.workerStopped()
	for request := range service.jobs {
		service.handle(request)
	}
}

func (service *Service) workerStopped() {
	service.workersMu.Lock()
	defer service.workersMu.Unlock()
	if service.remainingWorkers == 0 {
		return
	}
	service.remainingWorkers--
	if service.remainingWorkers == 0 {
		close(service.workersDone)
	}
}

func (service *Service) handle(request *Request) {
	// The error response is needed only on failure. Building it for every
	// successful request would allocate its headers and body on the hot path.
	var response Response
	if request.Body != nil {
		defer func() { _ = request.Body.Close() }()
	}
	service.handlersMu.RLock()
	handler := service.handlers[request.HandlerBindingID]
	service.handlersMu.RUnlock()
	if handler == nil {
		service.errors.add(errors.New("request has no registered Go handler"))
		_ = service.runtime.Respond(request.Exchange, internalResponse())
		return
	}
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				service.errors.add(fmt.Errorf("handler panic: %v", recovered))
				response = internalResponse()
			}
		}()
		value, err := handler(request)
		if err != nil {
			service.errors.add(err)
			response = internalResponse()
			return
		}
		response = value
	}()
	if response.sse != nil {
		service.handleSSEResponse(request.Exchange, response)
		return
	}
	if response.file != nil {
		if err := service.runtime.RespondFile(request.Exchange, *response.file); err != nil {
			if !retiredSubmission(err) {
				service.errors.add(err)
			}
		}
		return
	}
	if response.webSocket != nil {
		if service.webSockets == nil {
			service.errors.add(errors.New("handler returned an undeclared WebSocket response"))
			_ = service.runtime.Respond(request.Exchange, internalResponse())
			return
		}
		if err := service.webSockets.accept(request.Exchange, response.webSocket); err != nil {
			if !retiredSubmission(err) {
				service.errors.add(err)
				_ = service.runtime.Respond(request.Exchange, internalResponse())
			}
		}
		return
	}
	if response.producer != nil {
		service.handleStreamingResponse(request.Exchange, response)
		return
	}
	if err := service.runtime.Respond(request.Exchange, response); err != nil {
		if retiredSubmission(err) {
			return
		}
		service.errors.add(err)
		fallback := internalResponse()
		if response.Status != fallback.Status || string(response.body) != string(fallback.body) {
			if fallbackErr := service.runtime.Respond(request.Exchange, fallback); fallbackErr != nil {
				if !retiredSubmission(fallbackErr) {
					service.errors.add(fallbackErr)
				}
			}
		}
	}
}

func internalResponse() Response {
	response, err := Text(500, "internal server error")
	if err != nil {
		panic("static internal response is invalid")
	}
	return response
}

func overloadedResponse() Response {
	response, err := Text(503, "service unavailable")
	if err != nil {
		panic("static overloaded response is invalid")
	}
	return response
}

func waitForServiceStage(done <-chan struct{}, deadline time.Time, stage string) error {
	select {
	case <-done:
		return nil
	default:
	}
	remaining := time.Until(deadline)
	if remaining <= 0 {
		return fmt.Errorf("service shutdown timed out waiting for %s", stage)
	}
	timer := time.NewTimer(remaining)
	defer timer.Stop()
	select {
	case <-done:
		return nil
	case <-timer.C:
		return fmt.Errorf("service shutdown timed out waiting for %s", stage)
	}
}

// Close stops admission, waits for Core's drain outcome and admitted Go
// handlers, then releases Runtime. It is idempotent and serialized. A caller
// timeout retains the owner for retry. Core expiry/abort/failure remains a
// DrainError even after cleanup succeeds; CloseError says whether it is retained.
func (service *Service) Close() error {
	if service == nil {
		return nil
	}
	service.closeMu.Lock()
	defer service.closeMu.Unlock()
	if service.closed {
		return service.shutdownErr
	}
	deadline := time.Now().Add(service.shutdownTimeout)
	if !service.shutdownStarted {
		service.shutdownStarted = true
		service.beginShutdown()
	}
	if !service.drainStarted {
		if err := service.runtime.Drain(); err != nil {
			return closeFailure(err, true)
		}
		service.drainStarted = true
	}
	if err := waitForServiceStage(service.pumpDone, deadline, "the event pump"); err != nil {
		return closeFailure(errors.Join(service.drainFailure(), err), true)
	}
	service.shutdownErr = service.drainFailure()
	if !service.jobsClosed {
		close(service.jobs)
		service.jobsClosed = true
	}
	if err := waitForServiceStage(service.workersDone, deadline, "handler workers"); err != nil {
		return closeFailure(errors.Join(service.shutdownErr, err), true)
	}
	if service.webSockets != nil {
		service.webSockets.stopReader()
		if err := waitForServiceStage(service.webSocketReaderDone, deadline, "the WebSocket reader"); err != nil {
			return closeFailure(errors.Join(service.shutdownErr, err), true)
		}
	}
	closeErr := service.runtime.Close()
	if service.webSockets != nil {
		service.webSockets.shutdown()
	}
	service.closed = !service.runtime.retained()
	outcome := closeFailure(errors.Join(service.shutdownErr, closeErr), !service.closed)
	if service.closed {
		service.shutdownErr = outcome
	}
	return outcome
}
