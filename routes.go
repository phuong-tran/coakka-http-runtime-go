package coakkahttp

// RouteState is one entry in a Core-issued route/binding cut. It contains no
// handler object, native pointer, matcher storage or request data.
type RouteState struct {
	// RouteID is the stable logical route identifier.
	RouteID uint64
	// HandlerBindingID identifies the captured binding in this snapshot.
	HandlerBindingID uint64
	// BindingRevision is the route's Core-issued binding revision.
	BindingRevision uint64
}

// RouteSnapshot is an application-owned copy of one coherent Core cut.
// Mutating this copy cannot change routing. It is control-plane observation,
// never a dispatch table; activation must still use Core's generation guards.
type RouteSnapshot struct {
	// RouteGeneration versions the complete structural route set.
	RouteGeneration uint64
	// MetadataGeneration versions the complete route-documentation set.
	MetadataGeneration uint64
	// BindingChangeSequence versions the binding cut across all entries.
	BindingChangeSequence uint64
	// Routes contains the complete cut, bounded by Core's accepted MaxRoutes.
	Routes []RouteState
}

// Routes returns a bounded complete copy, independently of monitor collection.
// Unresolved control or any failed read returns an error and no partial snapshot.
func (runtime *Runtime) Routes() (RouteSnapshot, error) {
	native, done, err := runtime.acquire()
	if err != nil {
		return RouteSnapshot{}, err
	}
	defer done.release()
	return native.routes()
}

// Routes reads Core truth, not the service's local prepared handler registry.
func (service *Service) Routes() (RouteSnapshot, error) {
	if service == nil || service.runtime == nil {
		return RouteSnapshot{}, errRuntimeClosed
	}
	return service.runtime.Routes()
}
