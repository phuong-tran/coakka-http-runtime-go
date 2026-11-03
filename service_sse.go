package coakkahttp

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// SSEProducer writes a finite sequence of server-sent events. The Service
// invokes it on one fixed handler worker; the writer must not escape or be used
// concurrently. Returning completes the response with an empty trailer block.
type SSEProducer func(*SSEWriter) error

// SSEWriter writes typed events under the same bounded writable-credit law as
// an ordinary response stream.
type SSEWriter struct {
	service  *Service
	exchange ExchangeID
	state    *responseStreamState
	timeout  time.Duration
	mu       sync.Mutex
	closed   bool
	started  bool
}

// Write waits for initial readiness and retries only an exact queue-full event
// after Runtime publishes recovery readiness.
func (writer *SSEWriter) Write(event SSEEvent) error {
	if writer == nil || writer.service == nil || writer.state == nil {
		return errors.New("SSE writer is invalid")
	}
	writer.mu.Lock()
	defer writer.mu.Unlock()
	if writer.closed {
		return errors.New("SSE writer is closed")
	}
	if !writer.started {
		if err := writer.state.awaitWritable(writer.timeout); err != nil {
			return err
		}
		writer.started = true
	}
	for {
		err := writer.service.runtime.WriteSSE(writer.exchange, event)
		if err == nil {
			return nil
		}
		var runtimeErr *Error
		if !errors.As(err, &runtimeErr) || runtimeErr.Code != ResultCapacityExhausted {
			return err
		}
		if err = writer.state.awaitWritable(writer.timeout); err != nil {
			return err
		}
	}
}

func (writer *SSEWriter) close() {
	writer.mu.Lock()
	writer.closed = true
	writer.mu.Unlock()
}

func (service *Service) handleSSEResponse(exchange ExchangeID, response Response) {
	state := newResponseStreamState()
	if err := service.registerResponseStream(exchange, state); err != nil {
		service.errors.add(err)
		_ = service.runtime.Respond(exchange, internalResponse())
		return
	}
	defer service.unregisterResponseStream(exchange, state)

	if err := service.runtime.StartSSE(exchange, response.Headers); err != nil {
		if !retiredSubmission(err) {
			service.errors.add(err)
			_ = service.runtime.Respond(exchange, internalResponse())
		}
		return
	}

	writer := &SSEWriter{
		service:  service,
		exchange: exchange,
		state:    state,
		timeout:  service.responseWriteTimeout,
	}
	producerErr := invokeSSEProducer(response.sse, writer)
	writer.close()
	if producerErr != nil {
		if !retiredSubmission(producerErr) {
			service.errors.add(producerErr)
		}
	}
	if err := service.runtime.FinishResponseStream(exchange, Headers{}); err != nil {
		if !retiredSubmission(err) {
			service.errors.add(err)
		}
	}
}

func invokeSSEProducer(producer SSEProducer, writer *SSEWriter) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("SSE producer panic: %v", recovered)
		}
	}()
	return producer(writer)
}
