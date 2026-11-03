package coakkahttp

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

// ResponseStreamProducer writes a response incrementally and returns optional
// response trailers. The Service invokes it on one fixed handler worker. The
// producer and writer must not escape the callback or be used concurrently.
type ResponseStreamProducer func(*ResponseStreamWriter) (Headers, error)

// coreStreamEnd projects a pulled Core cancellation/terminal, not a guessed
// native result or a Go timeout. Only the event pump creates it. Keep it distinct
// from producer errors with identical text; io.ErrClosedPipe is the Go-facing
// stream meaning, not an invented HTTP failure reason.
type coreStreamEnd struct{ kind EventKind }

// Error describes the Core-issued end without inventing its transport cause.
func (ended *coreStreamEnd) Error() string {
	if ended.kind == EventRequestCancelled {
		return "Core cancelled the response stream request"
	}
	return "Core ended the response stream exchange"
}

// Unwrap exposes Go's closed-writer meaning while retaining the Core event type.
func (*coreStreamEnd) Unwrap() error { return io.ErrClosedPipe }

type responseStreamState struct {
	mu       sync.Mutex
	notify   chan struct{}
	writable bool
	terminal error
}

func newResponseStreamState() *responseStreamState {
	return &responseStreamState{notify: make(chan struct{}, 1)}
}

func (state *responseStreamState) wake() {
	select {
	case state.notify <- struct{}{}:
	default:
	}
}

func (state *responseStreamState) grantWritable() {
	state.mu.Lock()
	if state.terminal == nil {
		state.writable = true
	}
	state.mu.Unlock()
	state.wake()
}

func (state *responseStreamState) terminate(err error) {
	state.mu.Lock()
	if state.terminal == nil {
		state.terminal = err
	}
	state.mu.Unlock()
	state.wake()
}

func (state *responseStreamState) awaitWritable(timeout time.Duration) error {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		state.mu.Lock()
		if state.terminal != nil {
			err := state.terminal
			state.mu.Unlock()
			return err
		}
		if state.writable {
			state.writable = false
			state.mu.Unlock()
			return nil
		}
		state.mu.Unlock()

		select {
		case <-state.notify:
		case <-timer.C:
			return errors.New("response stream writable-credit wait timed out")
		}
	}
}

// ResponseStreamWriter writes copied chunks after Runtime's initial writability
// hint and waits again only after queue pressure. A writer is valid only while
// its producer callback runs.
type ResponseStreamWriter struct {
	service  *Service
	exchange ExchangeID
	state    *responseStreamState
	timeout  time.Duration
	mu       sync.Mutex
	closed   bool
	started  bool
}

// Write waits for initial credit, retries exact queue pressure only after a
// recovery credit, and copies the accepted chunk into Runtime.
func (writer *ResponseStreamWriter) Write(data []byte) (int, error) {
	if writer == nil || writer.service == nil || writer.state == nil {
		return 0, errors.New("response stream writer is invalid")
	}
	if len(data) == 0 {
		return 0, nil
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return 0, errors.New("response stream writer is closed")
	}
	if !writer.started {
		if err := writer.state.awaitWritable(writer.timeout); err != nil {
			return 0, err
		}
		writer.started = true
	}
	for {
		err := writer.service.runtime.WriteResponseStream(writer.exchange, data)
		if err == nil {
			return len(data), nil
		}
		var runtimeErr *Error
		if !errors.As(err, &runtimeErr) || runtimeErr.Code != ResultCapacityExhausted {
			return 0, err
		}
		if err = writer.state.awaitWritable(writer.timeout); err != nil {
			return 0, err
		}
	}
}

func (writer *ResponseStreamWriter) close() {
	writer.mu.Lock()
	writer.closed = true
	writer.mu.Unlock()
}

func (service *Service) registerResponseStream(exchange ExchangeID, state *responseStreamState) error {
	service.stateMu.Lock()
	defer service.stateMu.Unlock()
	if _, exists := service.responseStreams[exchange]; exists {
		return errors.New("response stream is already registered")
	}
	service.responseStreams[exchange] = state
	return nil
}

func (service *Service) unregisterResponseStream(exchange ExchangeID, state *responseStreamState) {
	service.stateMu.Lock()
	if service.responseStreams[exchange] == state {
		delete(service.responseStreams, exchange)
	}
	service.stateMu.Unlock()
}

func (service *Service) signalResponseStreamWritable(exchange ExchangeID) {
	service.stateMu.Lock()
	state := service.responseStreams[exchange]
	service.stateMu.Unlock()
	if state != nil {
		state.grantWritable()
	}
}

func (service *Service) signalResponseStreamTerminal(exchange ExchangeID, err error) {
	service.stateMu.Lock()
	state := service.responseStreams[exchange]
	service.stateMu.Unlock()
	if state != nil {
		state.terminate(err)
	}
}

func (service *Service) handleStreamingResponse(exchange ExchangeID, response Response) {
	state := newResponseStreamState()
	if err := service.registerResponseStream(exchange, state); err != nil {
		service.errors.add(err)
		_ = service.runtime.Respond(exchange, internalResponse())
		return
	}
	defer service.unregisterResponseStream(exchange, state)

	head := response
	head.producer = nil
	if err := service.runtime.StartResponseStream(exchange, head); err != nil {
		if !retiredSubmission(err) {
			service.errors.add(err)
			_ = service.runtime.Respond(exchange, internalResponse())
		}
		return
	}

	writer := &ResponseStreamWriter{
		service:  service,
		exchange: exchange,
		state:    state,
		timeout:  service.responseWriteTimeout,
	}
	trailers, producerErr := invokeResponseStreamProducer(response.producer, writer)
	writer.close()
	if producerErr != nil {
		if !retiredSubmission(producerErr) {
			service.errors.add(producerErr)
		}
		trailers = Headers{}
	}
	if err := service.runtime.FinishResponseStream(exchange, trailers); err != nil {
		if !retiredSubmission(err) {
			service.errors.add(err)
		}
	}
}

func invokeResponseStreamProducer(producer ResponseStreamProducer, writer *ResponseStreamWriter) (trailers Headers, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			trailers = Headers{}
			err = fmt.Errorf("response stream producer panic: %v", recovered)
		}
	}()
	return producer(writer)
}
