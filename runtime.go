package coakkahttp

import (
	"errors"
	"sync"
	"time"
)

var errRuntimeClosed = errors.New("CoAkka HTTP Runtime is closing or closed")

// Runtime is the lifecycle owner for one callback-free native HTTP runtime.
// Methods may be called concurrently. Lifecycle transitions are serialized,
// and each Take lane admits one reader at a time. Close interrupts interruptible
// readers and waits for every acquired call before destroying the native service.
type Runtime struct {
	mu           sync.Mutex
	condition    *sync.Cond
	lifecycle    sync.Mutex
	native       *nativeService
	dispatcher   *dispatcher
	active       uint32
	closing      bool
	closeStarted bool
	started      bool
	closeErr     error

	eventReader     sync.Mutex
	webSocketReader sync.Mutex
	outboundReader  sync.Mutex
	monitorWaiter   sync.Mutex
}

// Features verifies the directly linked host ABI and returns the loaded Core's
// immutable capability matrix without a service or OS probe.
func Features() (Capabilities, error) {
	api, err := openNativeAPI()
	if err != nil {
		return 0, err
	}
	capabilities := nativeFeatures(api)
	closeNativeAPI(&api)
	return capabilities, nil
}

// ServicePlan is the immutable capability check for one complete service.
type ServicePlan struct {
	// Supported reports whether the connector can execute the declaration.
	Supported bool
	// RequiredCapabilities is the dependency-normalized requirement mask.
	RequiredCapabilities Capabilities
	// AdapterCapabilities is the connector implementation mask.
	AdapterCapabilities Capabilities
	// MissingCapabilities identifies requirements absent from the adapter.
	MissingCapabilities Capabilities
}

// PlanService validates a complete declaration. adapterCapabilities contains
// connector implementation facts only; native owns every platform probe.
func PlanService(requiredCapabilities, adapterCapabilities Capabilities) (ServicePlan, error) {
	api, err := openNativeAPI()
	if err != nil {
		return ServicePlan{}, err
	}
	defer closeNativeAPI(&api)
	return nativePlanService(api, requiredCapabilities, adapterCapabilities)
}

// RuntimeInspection contains Core image/probe facts without inventing service
// configuration. InspectRuntime explicitly probes; per-service reads do not.
type RuntimeInspection struct {
	// ABIVersion identifies the loaded public Runtime contract.
	ABIVersion uint32
	// Features contains the immutable Runtime feature bits.
	Features Capabilities
	// IOUringCompiled reports whether this Runtime image includes io_uring support.
	IOUringCompiled bool
	// IOUringProbed distinguishes unprobed from probed-but-unsupported.
	IOUringProbed bool
	// IOUringSupported is meaningful when IOUringProbed is true.
	IOUringSupported bool
	// IOUringProbeError preserves the native refusal diagnostic, or zero.
	IOUringProbeError int32
}

// RuntimeInfo is a pointer-free snapshot copied from Core plus the Core-issued
// immutable dispatcher placement. Reading it performs no additional OS probe.
type RuntimeInfo struct {
	RuntimeInspection
	// Limits is Core's immutable accepted startup configuration, including
	// resolved defaults. It is not reconstructed from the Go Config input.
	Limits RuntimeLimits
	// CPU is Core's entry-time dispatcher/child-thread placement, not utilization.
	CPU CPUInfo
	// Execution is a cold Core observation of configured and live HTTP loops;
	// it is independent of CPU placement and Go worker/GOMAXPROCS settings.
	Execution ExecutionInfo
	// RequestedIOBackend is the preference passed to this Core service.
	RequestedIOBackend IOBackend
	// EffectiveIOBackend is the backend selected by Core for this service.
	EffectiveIOBackend IOBackend
	// FallbackReason explains a requested/effective backend difference.
	FallbackReason IOBackendFallbackReason
	// ServiceStarted reports whether the inspected Runtime is currently started.
	ServiceStarted bool
	// RequestNotificationBatchSize is Core's selected HTTP/1 notification cap.
	RequestNotificationBatchSize uint32
	// TerminalNotificationBatchSize is Core's selected terminal doorbell cap
	// before queue-capacity clamping.
	TerminalNotificationBatchSize uint32
	// EventLoopCPUMask is the requested logical CPU set; zero inherits OS placement.
	EventLoopCPUMask uint64
	// EventLoopCPUMaskSupported reports whether this native build accepts explicit masks.
	EventLoopCPUMaskSupported bool
	// EventLoopCPUMaskApplied reports verified explicit placement at startup, not a live OS read.
	EventLoopCPUMaskApplied bool
}

// InspectRuntime explicitly asks Core to probe optional backend support. It
// creates no service and reports no selected CPU, batch or effective backend.
func InspectRuntime() (RuntimeInspection, error) {
	api, err := openNativeAPI()
	if err != nil {
		return RuntimeInspection{}, err
	}
	defer closeNativeAPI(&api)
	return nativeRuntimeInfo(api)
}

// Open validates and copies a complete Config into a new, unstarted Runtime.
func Open(config Config) (*Runtime, error) {
	dispatcher, native, err := openDispatcher(config)
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{native: native, dispatcher: dispatcher}
	runtime.condition = sync.NewCond(&runtime.mu)
	return runtime, nil
}

// runtimeCall is one admitted call, released exactly once by its caller's
// defer. A value token avoids a heap closure without weakening Close's active
// call barrier. Do not copy a token into another owner or release it twice.
type runtimeCall struct {
	owner *Runtime
}

func (call runtimeCall) release() {
	runtime := call.owner
	runtime.mu.Lock()
	runtime.active--
	if runtime.active == 0 {
		runtime.condition.Broadcast()
	}
	runtime.mu.Unlock()
}

func (runtime *Runtime) acquire() (*nativeService, runtimeCall, error) {
	if runtime == nil {
		return nil, runtimeCall{}, errors.New("CoAkka HTTP Runtime is nil")
	}
	runtime.mu.Lock()
	if runtime.native == nil || runtime.closeStarted {
		runtime.mu.Unlock()
		return nil, runtimeCall{}, errRuntimeClosed
	}
	runtime.active++
	native := runtime.native
	runtime.mu.Unlock()
	return native, runtimeCall{owner: runtime}, nil
}

// retained reads Core's checked-destroy outcome, never guesses from an error
// code: destruction may succeed before a separate CPU restoration failure.
func (runtime *Runtime) retained() bool {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	return runtime.native != nil
}

// Capabilities returns the immutable feature-bit matrix of the loaded Runtime.
func (runtime *Runtime) Capabilities() Capabilities {
	if runtime == nil {
		return 0
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if runtime.native == nil {
		return 0
	}
	return runtime.native.capabilities
}

// RuntimeInfo reports capability and requested/effective backend truth.
func (runtime *Runtime) RuntimeInfo() (RuntimeInfo, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return RuntimeInfo{}, err
	}
	defer done.release()
	info, err := native.runtimeInfo()
	info.CPU = runtime.dispatcher.cpu
	return info, err
}

// Start binds configured listeners and starts native admission.
func (runtime *Runtime) Start() error {
	if runtime == nil {
		return errors.New("CoAkka HTTP Runtime is nil")
	}
	runtime.lifecycle.Lock()
	defer runtime.lifecycle.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	runtime.dispatcher.call(func() { err = native.start() })
	if err != nil {
		return err
	}
	runtime.mu.Lock()
	runtime.started = true
	runtime.mu.Unlock()
	return nil
}

// BoundPort returns Runtime's selected port for a started single-listener setup.
func (runtime *Runtime) BoundPort() (uint16, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return 0, err
	}
	defer done.release()
	return native.boundPort()
}

// Drain begins Core's bounded graceful drain. It does not wait for completion;
// use WaitDrained while continuing to serve admitted work.
func (runtime *Runtime) Drain() error {
	if runtime == nil {
		return errors.New("CoAkka HTTP Runtime is nil")
	}
	runtime.lifecycle.Lock()
	defer runtime.lifecycle.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.drain()
}

// Stop performs Runtime's bounded idempotent stop.
func (runtime *Runtime) Stop() error {
	if runtime == nil {
		return errors.New("CoAkka HTTP Runtime is nil")
	}
	runtime.lifecycle.Lock()
	defer runtime.lifecycle.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	err = native.stop()
	if err == nil {
		runtime.mu.Lock()
		runtime.started = false
		runtime.mu.Unlock()
	}
	return err
}

// Interrupt wakes the sole inbound event reader without fabricating an event.
func (runtime *Runtime) Interrupt() error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.interrupt()
}

// Close interrupts readers, waits for all native calls and automatic lease
// releases, then stops, destroys, and unloads. This is forced resource cleanup,
// not a graceful-drain certificate. For graceful shutdown call Drain, continue
// serving admitted work, and wait for WaitDrained before Close (or use Service).
// Concurrent calls share
// the same terminal result; a failed destroy remains retryable. A CloseError
// reports whether Core remains retained, including errors after destruction.
func (runtime *Runtime) Close() error {
	if runtime == nil {
		return nil
	}
	// Runtime's public C ABI forbids overlapping lifecycle transitions. Holding
	// this lock before reading started state also prevents Close from racing a
	// Start that has crossed the native boundary but has not published success.
	runtime.lifecycle.Lock()
	defer runtime.lifecycle.Unlock()
	runtime.mu.Lock()
	for runtime.closing {
		runtime.condition.Wait()
	}
	if runtime.native == nil {
		err := runtime.closeErr
		runtime.mu.Unlock()
		return err
	}
	runtime.closing = true
	runtime.closeStarted = true
	native := runtime.native
	started := runtime.started
	runtime.mu.Unlock()

	// These wakeups are specifically defined to overlap the corresponding sole
	// readers. Other Take lanes are finite-bounded by nativeDuration.
	_ = native.interrupt()
	_ = native.interruptOutbound()
	_ = native.monitorInterrupt()
	runtime.mu.Lock()
	for runtime.active != 0 {
		runtime.condition.Wait()
	}
	runtime.mu.Unlock()

	var err error
	if started {
		stopErr := native.stop()
		err = errors.Join(err, stopErr)
		if stopErr == nil {
			runtime.mu.Lock()
			runtime.started = false
			runtime.mu.Unlock()
		}
		// Do not attempt stop a second time through destroy after refusal.
		// Keep the reachable owner and reject ordinary operations until retry.
		if stopErr != nil {
			err = closeFailure(err, true)
			runtime.mu.Lock()
			runtime.closeErr = err
			runtime.closing = false
			runtime.condition.Broadcast()
			runtime.mu.Unlock()
			return err
		}
	}
	err = errors.Join(err, native.destroy())
	if native.service == nil {
		err = errors.Join(err, runtime.dispatcher.close())
	}
	err = closeFailure(err, native.service != nil)

	runtime.mu.Lock()
	if native.service == nil {
		runtime.native = nil
		runtime.started = false
	}
	runtime.closeErr = err
	runtime.closing = false
	runtime.condition.Broadcast()
	runtime.mu.Unlock()
	return err
}

// TakeEvent waits for and copies one inbound event. The native event lease is
// released exactly once before this method returns, including copy failures.
func (runtime *Runtime) TakeEvent(timeout time.Duration) (*Event, error) {
	if runtime == nil {
		return nil, errRuntimeClosed
	}
	value, err := nativeDuration(timeout)
	if err != nil {
		return nil, err
	}
	runtime.eventReader.Lock()
	defer runtime.eventReader.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return nil, err
	}
	defer done.release()
	var event *Event
	runtime.dispatcher.call(func() { event, err = native.takeEvent(value) })
	return event, err
}

// takeDispatcherEvent is exclusively called by Service's pump already running
// on the locked dispatcher. Avoid a second thread/channel handoff per request.
func (runtime *Runtime) takeDispatcherEvent(timeout time.Duration) (*Event, error) {
	value, err := nativeDuration(timeout)
	if err != nil {
		return nil, err
	}
	native, done, err := runtime.acquire()
	if err != nil {
		return nil, err
	}
	defer done.release()
	return native.takeEvent(value)
}

// ExchangeCancelled reports whether Runtime has observed cancellation.
func (runtime *Runtime) ExchangeCancelled(exchange ExchangeID) (bool, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return false, err
	}
	defer done.release()
	return native.exchangeCancelled(exchange)
}

// Respond completes an exchange with one copied buffered response.
func (runtime *Runtime) Respond(exchange ExchangeID, response Response) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.respond(exchange, response)
}

// RespondFile completes an exchange with a Runtime-opened authority-scoped file.
func (runtime *Runtime) RespondFile(exchange ExchangeID, response FileResponse) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.respondFile(exchange, response)
}

// Fail completes an exchange with a typed copied failure.
func (runtime *Runtime) Fail(exchange ExchangeID, failure Failure) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.fail(exchange, failure)
}

// StartResponseStream sends a copied response head and opens a stream.
func (runtime *Runtime) StartResponseStream(exchange ExchangeID, head Response) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.startResponseStream(exchange, head)
}

// StartSSE sends a Runtime-defined SSE response head with copied extra headers.
func (runtime *Runtime) StartSSE(exchange ExchangeID, headers Headers) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.startSSE(exchange, headers)
}

// WriteResponseStream copies one bounded response body chunk into Runtime.
func (runtime *Runtime) WriteResponseStream(exchange ExchangeID, data []byte) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.writeResponseStream(exchange, data)
}

// WriteSSE copies one typed server-sent event into Runtime.
func (runtime *Runtime) WriteSSE(exchange ExchangeID, event SSEEvent) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.writeSSE(exchange, event)
}

// FinishResponseStream completes a stream with copied trailers.
func (runtime *Runtime) FinishResponseStream(exchange ExchangeID, trailers Headers) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.finishResponseStream(exchange, trailers)
}

// AcceptWebSocket accepts an upgrade and copies the selected subprotocol.
func (runtime *Runtime) AcceptWebSocket(exchange ExchangeID, selectedSubprotocol string) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.acceptWebSocket(exchange, selectedSubprotocol)
}

// TakeWebSocket waits for and copies one socket event, releasing the native
// lease exactly once before returning.
func (runtime *Runtime) TakeWebSocket(timeout time.Duration) (*SocketEvent, error) {
	value, err := nativeDuration(timeout)
	if err != nil {
		return nil, err
	}
	runtime.webSocketReader.Lock()
	defer runtime.webSocketReader.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return nil, err
	}
	defer done.release()
	return native.takeWebSocket(value)
}

// SendWebSocket copies one text, binary, ping, or pong frame into Runtime.
func (runtime *Runtime) SendWebSocket(session WebSocketID, kind SocketEventKind, data []byte) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.sendWebSocket(session, kind, data)
}

// CloseWebSocket asks Runtime to close a session with a copied reason.
func (runtime *Runtime) CloseWebSocket(session WebSocketID, code uint32, reason string) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.closeWebSocket(session, code, reason)
}

// SubmitOutbound copies and admits one Runtime-owned HTTP client call.
func (runtime *Runtime) SubmitOutbound(request ClientRequest) (OutboundID, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return OutboundID{}, err
	}
	defer done.release()
	return native.submitOutbound(request)
}

// CancelOutbound requests cancellation of one outbound call.
func (runtime *Runtime) CancelOutbound(call OutboundID) error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.cancelOutbound(call)
}

// TakeOutbound waits for and copies one outbound terminal, releasing the
// native terminal lease exactly once before returning.
func (runtime *Runtime) TakeOutbound(timeout time.Duration) (*ClientTerminal, error) {
	value, err := nativeDuration(timeout)
	if err != nil {
		return nil, err
	}
	runtime.outboundReader.Lock()
	defer runtime.outboundReader.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return nil, err
	}
	defer done.release()
	return native.takeOutbound(value)
}

// Rebind applies a generation-checked handler-binding update using Core's
// bounded observation budget. Timeout/wakeup failure is ambiguous: retry the
// exact activation/payload. An outcome's retirement proof is independent.
func (runtime *Runtime) Rebind(request RebindRequest) (RebindOutcome, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return RebindOutcome{}, err
	}
	defer done.release()
	return native.rebind(request)
}

// PublishRoutes applies one complete generation of structural route truth.
// Timeout and wakeup-failed outcomes are delivery-ambiguous; retry the exact
// activation and route payload to resolve the operation idempotently.
// Core owns the observation budget; there is no connector timeout override.
func (runtime *Runtime) PublishRoutes(request RoutePublication) (RoutePublicationOutcome, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return RoutePublicationOutcome{}, err
	}
	defer done.release()
	return native.publishRoutes(request)
}

// Health returns a non-blocking copied health snapshot.
func (runtime *Runtime) Health() (Health, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return Health{}, err
	}
	defer done.release()
	return native.health()
}

// ProbeLiveness requests and waits for fresh Runtime progress evidence.
func (runtime *Runtime) ProbeLiveness(timeout time.Duration) (Health, error) {
	value, err := nativeDuration(timeout)
	if err != nil {
		return Health{}, err
	}
	native, done, err := runtime.acquire()
	if err != nil {
		return Health{}, err
	}
	defer done.release()
	return native.probeLiveness(value)
}

// MonitorConfig returns the copied effective monitoring configuration.
func (runtime *Runtime) MonitorConfig() (MonitorConfig, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return MonitorConfig{}, err
	}
	defer done.release()
	return native.monitorConfig()
}

// MonitorSnapshot returns copied bounded aggregates and co-observed health.
func (runtime *Runtime) MonitorSnapshot() (MonitorSnapshot, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return MonitorSnapshot{}, err
	}
	defer done.release()
	return native.monitorSnapshot()
}

// ApplyMonitorPolicy performs an optimistic live monitor policy update.
func (runtime *Runtime) ApplyMonitorPolicy(expectedGeneration uint64, policy MonitorPolicy) (MonitorApplyOutcome, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return MonitorApplyOutcome{}, err
	}
	defer done.release()
	return native.monitorApply(expectedGeneration, policy)
}

// ReadMonitorEvents reads one bounded cursor page of copied monitor events.
func (runtime *Runtime) ReadMonitorEvents(afterSequence uint64, maximumEvents uint32) (MonitorEventPage, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return MonitorEventPage{}, err
	}
	defer done.release()
	return native.monitorRead(afterSequence, maximumEvents)
}

// WaitMonitor waits until newer monitoring truth may be available.
func (runtime *Runtime) WaitMonitor(timeout time.Duration) error {
	value, err := nativeDuration(timeout)
	if err != nil {
		return err
	}
	runtime.monitorWaiter.Lock()
	defer runtime.monitorWaiter.Unlock()
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.monitorWait(value)
}

// InterruptMonitor wakes the sole monitor waiter without fabricating data.
func (runtime *Runtime) InterruptMonitor() error {
	native, done, err := runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return native.monitorInterrupt()
}
