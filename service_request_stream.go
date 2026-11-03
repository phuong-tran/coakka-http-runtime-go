package coakkahttp

import (
	"errors"
	"io"
	"sync"
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
}

// requestBodyStream has one pump-side producer and one handler-side reader.
// Runtime event copies are retained in a construction-bounded channel; terminal
// state owns no extra queue slot and closing wakes a blocked Read.
type requestBodyStream struct {
	mu          sync.Mutex
	events      chan requestBodyEvent
	request     *Request
	closed      bool
	terminalErr error
	expected    uint64
	current     []byte
	offset      int
}

func newRequestBodyStream(request *Request, capacity uint32, initialSequence uint64) *requestBodyStream {
	return &requestBodyStream{
		events:   make(chan requestBodyEvent, int(capacity)),
		request:  request,
		expected: initialSequence + 1,
	}
}

func (stream *requestBodyStream) offer(event requestBodyEvent) bool {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if stream.closed {
		return true
	}
	if event.sequence != stream.expected {
		stream.closed = true
		stream.terminalErr = errRequestBodySequence
		close(stream.events)
		return false
	}
	stream.expected++
	if event.kind == EventRequestEnd {
		stream.closed = true
		close(stream.events)
		return true
	}
	select {
	case stream.events <- event:
		return true
	default:
		stream.closed = true
		stream.terminalErr = errRequestBodyOverflow
		close(stream.events)
		return false
	}
}

func (stream *requestBodyStream) finish(err error) {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	if stream.closed {
		return
	}
	stream.closed = true
	stream.terminalErr = err
	close(stream.events)
}

func (stream *requestBodyStream) terminalError() error {
	stream.mu.Lock()
	defer stream.mu.Unlock()
	return stream.terminalErr
}

// Read implements io.Reader for the sole handler-side request-body consumer.
func (stream *requestBodyStream) Read(target []byte) (int, error) {
	if len(target) == 0 {
		return 0, nil
	}
	for {
		if stream.offset < len(stream.current) {
			count := copy(target, stream.current[stream.offset:])
			stream.offset += count
			if stream.offset == len(stream.current) {
				stream.current = nil
				stream.offset = 0
			}
			return count, nil
		}
		event, ok := <-stream.events
		if !ok {
			if err := stream.terminalError(); err != nil {
				return 0, err
			}
			return 0, io.EOF
		}
		switch event.kind {
		case EventRequestData:
			stream.current = event.data
		case EventRequestTrailers:
			stream.request.Trailers = event.trailers
		}
	}
}

// Close interrupts a blocked Read and releases the handler's local reader.
func (stream *requestBodyStream) Close() error {
	stream.finish(io.ErrClosedPipe)
	return nil
}

func (service *Service) routeRequestStreamEvent(event *Event) {
	service.stateMu.Lock()
	stream := service.requestStreams[event.Exchange]
	if event.Kind == EventRequestEnd {
		delete(service.requestStreams, event.Exchange)
	}
	service.stateMu.Unlock()
	if stream == nil {
		service.errors.add(errors.New("request body event has no active stream"))
		return
	}
	accepted := stream.offer(requestBodyEvent{
		kind:     event.Kind,
		sequence: event.Sequence,
		data:     event.Data,
		trailers: event.StreamTrailers,
	})
	if !accepted {
		service.errors.add(errRequestBodyOverflow)
		if err := service.runtime.Respond(event.Exchange, internalResponse()); err != nil {
			service.errors.add(err)
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
