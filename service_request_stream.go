package coakkahttp

import (
	"errors"
	"io"
	"sync"
	"sync/atomic"
)

var (
	errRequestBodyCancelled  = errors.New("request body stream was cancelled")
	errRequestBodyOverflow   = errors.New("request body stream queue is full")
	errRequestBodySequence   = errors.New("request body stream sequence is discontinuous")
	errRequestBodyTerminated = errors.New("request body stream reached terminal state")
)

type requestBodyEvent struct {
	kind     EventKind
	sequence uint64
	data     []byte
	trailers Headers
	lease    *requestPayloadLease
}

// requestBodyStream has one producer and one handler-side reader. The lock
// makes taking an item, consuming it and discarding it mutually exclusive with
// Close, so no lease can be in transit when the dispatcher tears down a stream.
type requestBodyStream struct {
	mu          sync.Mutex
	available   sync.Cond
	events      []requestBodyEvent
	head        int
	count       int
	request     *Request
	returns     *requestPayloadReturns
	ended       bool
	closed      bool
	terminalErr error
	expected    uint64
	offset      int
}

func newRequestBodyStream(request *Request, capacity uint32, initialSequence uint64) *requestBodyStream {
	stream := &requestBodyStream{
		events:   make([]requestBodyEvent, capacity),
		request:  request,
		expected: initialSequence + 1,
	}
	stream.available.L = &stream.mu
	return stream
}

func (stream *requestBodyStream) offer(event requestBodyEvent) error {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if stream.closed || stream.ended {
		stream.returnPayload(event)
		return nil
	}
	if event.sequence != stream.expected {
		stream.terminalErr = errRequestBodySequence
	} else if event.kind == EventRequestEnd {
		stream.ended = true
		stream.available.Signal()
		return nil
	} else if stream.count == len(stream.events) {
		// Core still owns credit for the item being read and every queued
		// item. This is an invariant guard for an inconsistent event stream.
		stream.terminalErr = errRequestBodyOverflow
	} else {
		tail := (stream.head + stream.count) % len(stream.events)
		stream.events[tail] = event
		stream.count++
		stream.expected++
		stream.available.Signal()
		return nil
	}
	stream.ended = true
	stream.returnPayload(event)
	stream.available.Signal()
	return stream.terminalErr
}

// finish discards the unread suffix before the dispatcher can exit. A complete
// reader keeps EOF, and a prior error remains the first failure it observed.
func (stream *requestBodyStream) finish(err error) {
	stream.mu.Lock()
	if stream.closed {
		stream.mu.Unlock()
		return
	}
	if stream.terminalErr == nil && (!stream.ended || stream.count != 0) {
		stream.terminalErr = err
	}
	stream.closed = true
	wake := false
	for stream.count != 0 {
		wake = stream.consume() || wake
	}
	stream.available.Signal()
	stream.mu.Unlock()
	if wake {
		stream.returns.wakeReader()
	}
}

func (stream *requestBodyStream) consume() bool {
	event := stream.events[stream.head]
	stream.events[stream.head] = requestBodyEvent{}
	stream.head = (stream.head + 1) % len(stream.events)
	stream.count--
	stream.offset = 0
	return stream.returnPayload(event)
}

func (stream *requestBodyStream) returnPayload(event requestBodyEvent) bool {
	return event.lease != nil && stream.returns.complete(event.lease)
}

// Read copies data into the handler buffer before acknowledging the Core lease.
func (stream *requestBodyStream) Read(target []byte) (int, error) {
	if len(target) == 0 {
		return 0, nil
	}
	wake := false
	stream.mu.Lock()
	defer func() {
		stream.mu.Unlock()
		if wake {
			stream.returns.wakeReader()
		}
	}()
	for {
		if stream.count != 0 {
			event := &stream.events[stream.head]
			if event.kind == EventRequestData {
				count := copy(target, event.data[stream.offset:])
				stream.offset += count
				if stream.offset == len(event.data) {
					wake = stream.consume() || wake
				}
				if count != 0 {
					return count, nil
				}
			} else {
				stream.request.Trailers = event.trailers
				wake = stream.consume() || wake
			}
			continue
		}
		if stream.closed || stream.ended {
			if stream.terminalErr != nil {
				return 0, stream.terminalErr
			}
			return 0, io.EOF
		}
		// Parsing TRAILERS can free the last credit required for END. Make
		// the return visible to the dispatcher before this reader sleeps.
		if wake {
			stream.mu.Unlock()
			stream.returns.wakeReader()
			stream.mu.Lock()
			wake = false
			continue
		}
		stream.available.Wait()
	}
}

// Close interrupts a blocked Read and returns unread native credits.
func (stream *requestBodyStream) Close() error {
	stream.finish(io.ErrClosedPipe)
	return nil
}

func (service *Service) routeRequestStreamEvent(event *Event, lease *requestPayloadLease) {
	service.stateMu.Lock()
	stream := service.requestStreams[event.Exchange]
	service.stateMu.Unlock()
	if stream == nil {
		if lease != nil {
			service.requestPayloads.complete(lease)
		}
		service.errors.add(errors.New("request body event has no active stream"))
		return
	}
	if err := stream.offer(requestBodyEvent{
		kind:     event.Kind,
		sequence: event.Sequence,
		data:     event.Data,
		trailers: event.StreamTrailers,
		lease:    lease,
	}); err != nil {
		service.errors.add(err)
		if responseErr := service.runtime.Respond(event.Exchange, internalResponse()); responseErr != nil {
			service.errors.add(responseErr)
		}
	}
}

func (service *Service) finishRequestStream(exchange ExchangeID, err error) {
	service.stateMu.Lock()
	stream := service.requestStreams[exchange]
	delete(service.requestStreams, exchange)
	service.stateMu.Unlock()
	if stream != nil {
		stream.finish(err)
	}
}

func (service *Service) finishRequestStreams() {
	service.stateMu.Lock()
	streams := service.requestStreams
	service.requestStreams = nil
	service.stateMu.Unlock()
	for _, stream := range streams {
		stream.finish(errRequestBodyTerminated)
	}
}

// requestPayloadReturns links existing lease descriptors. Its total size is
// bounded by the payload leases issued by Core. Only the dispatcher removes a
// node, and it does so only after Core accepts the release.
type requestPayloadReturns struct {
	mu    sync.Mutex
	first *requestPayloadLease
	last  *requestPayloadLease
	ready atomic.Bool
	wake  func()
}

func (queue *requestPayloadReturns) complete(lease *requestPayloadLease) bool {
	queue.mu.Lock()
	wasEmpty := queue.first == nil
	if wasEmpty {
		queue.first = lease
		queue.ready.Store(true)
	} else {
		queue.last.next = lease
	}
	queue.last = lease
	queue.mu.Unlock()
	return wasEmpty
}

func (queue *requestPayloadReturns) wakeReader() {
	if queue.wake != nil {
		queue.wake()
	}
}

func (queue *requestPayloadReturns) flush(release func(*requestPayloadLease) error) error {
	for {
		queue.mu.Lock()
		lease := queue.first
		queue.mu.Unlock()
		if lease == nil {
			return nil
		}
		if err := release(lease); err != nil {
			return err
		}
		queue.mu.Lock()
		queue.first = lease.next
		lease.next = nil
		if queue.first == nil {
			queue.last = nil
			queue.ready.Store(false)
		}
		queue.mu.Unlock()
	}
}

func (service *Service) flushRequestPayloads() error {
	if !service.requestPayloads.ready.Load() {
		return nil
	}
	native, done, err := service.runtime.acquire()
	if err != nil {
		return err
	}
	defer done.release()
	return service.requestPayloads.flush(native.releaseRequestPayload)
}

func (service *Service) wakeRequestPayloadReader() {
	err := service.runtime.Interrupt()
	var nativeErr *Error
	if err != nil && !errors.Is(err, errRuntimeClosed) &&
		!(errors.As(err, &nativeErr) && nativeErr.Code == ResultClosed) {
		service.errors.add(err)
	}
}
