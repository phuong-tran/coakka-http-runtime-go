package coakkahttp

import (
	"errors"
	"fmt"
	"sync"
)

// WebSocketHandler receives copied Runtime events in session order. It runs on
// the Service's sole WebSocket reader and must return promptly. An error records
// a bounded diagnostic and closes the session with status 1011.
type WebSocketHandler func(*WebSocketSession, SocketEvent) error

type webSocketPlan struct {
	subprotocol string
	handler     WebSocketHandler
}

// NewWebSocketResponse creates a WebSocket upgrade selection for a route
// registered with Builder.WebSocket or Builder.HandleWebSocket.
func NewWebSocketResponse(subprotocol string, handler WebSocketHandler) (Response, error) {
	if handler == nil || len(subprotocol) > 255 {
		return Response{}, errors.New("WebSocket response is invalid")
	}
	for index := 0; index < len(subprotocol); index++ {
		if subprotocol[index] > 0x7f || !isTokenByte(subprotocol[index]) {
			return Response{}, errors.New("WebSocket subprotocol is invalid")
		}
	}
	return Response{Status: 101, webSocket: &webSocketPlan{subprotocol: subprotocol, handler: handler}}, nil
}

func isTokenByte(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' ||
		value >= 'a' && value <= 'z' ||
		value == '!' || value == '#' || value == '$' || value == '%' ||
		value == '&' || value == '\'' || value == '*' || value == '+' ||
		value == '-' || value == '.' || value == '^' || value == '_' ||
		value == '`' || value == '|' || value == '~'
}

// WebSocketSession is a generation-safe session owner passed to callbacks.
type WebSocketSession struct {
	runtime     *Runtime
	id          WebSocketID
	subprotocol string
	mu          sync.Mutex
	open        bool
}

// Subprotocol returns the protocol selected for this session.
func (session *WebSocketSession) Subprotocol() string { return session.subprotocol }

// Open reports whether the Service still owns an open session.
func (session *WebSocketSession) Open() bool {
	if session == nil {
		return false
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	return session.open
}

// SendText copies one UTF-8 text frame into Runtime.
func (session *WebSocketSession) SendText(value string) error {
	return session.send(WebSocketText, []byte(value))
}

// SendBinary copies one binary frame into Runtime.
func (session *WebSocketSession) SendBinary(value []byte) error {
	return session.send(WebSocketBinary, value)
}

// Ping copies one ping payload into Runtime.
func (session *WebSocketSession) Ping(value []byte) error { return session.send(WebSocketPing, value) }

// Pong copies one pong payload into Runtime.
func (session *WebSocketSession) Pong(value []byte) error { return session.send(WebSocketPong, value) }

func (session *WebSocketSession) send(kind SocketEventKind, value []byte) error {
	if session == nil || session.runtime == nil {
		return errors.New("WebSocket session is invalid")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.open {
		return errors.New("WebSocket session is closed")
	}
	return session.runtime.SendWebSocket(session.id, kind, value)
}

// Close starts a close handshake. The session remains open until Runtime reports
// its terminal close event.
func (session *WebSocketSession) Close(code uint32, reason string) error {
	if session == nil || session.runtime == nil {
		return errors.New("WebSocket session is invalid")
	}
	session.mu.Lock()
	defer session.mu.Unlock()
	if !session.open {
		return nil
	}
	return session.runtime.CloseWebSocket(session.id, code, reason)
}

type webSocketEntry struct {
	session *WebSocketSession
	plan    *webSocketPlan
}

type webSocketDispatcher struct {
	service  *Service
	capacity uint32
	mu       sync.Mutex
	closed   bool
	pending  map[ExchangeID]*webSocketPlan
	sessions map[WebSocketID]webSocketEntry
}

func newWebSocketDispatcher(service *Service, capacity uint32) *webSocketDispatcher {
	return &webSocketDispatcher{
		service: service, capacity: capacity,
		pending:  make(map[ExchangeID]*webSocketPlan, capacity),
		sessions: make(map[WebSocketID]webSocketEntry, capacity),
	}
}

func (dispatcher *webSocketDispatcher) accept(exchange ExchangeID, plan *webSocketPlan) error {
	dispatcher.mu.Lock()
	if dispatcher.closed || uint32(len(dispatcher.pending)) >= dispatcher.capacity {
		dispatcher.mu.Unlock()
		return errors.New("WebSocket upgrade capacity is exhausted")
	}
	if _, exists := dispatcher.pending[exchange]; exists {
		dispatcher.mu.Unlock()
		return errors.New("WebSocket upgrade is already pending")
	}
	dispatcher.pending[exchange] = plan
	dispatcher.mu.Unlock()
	if err := dispatcher.service.runtime.AcceptWebSocket(exchange, plan.subprotocol); err != nil {
		dispatcher.mu.Lock()
		delete(dispatcher.pending, exchange)
		dispatcher.mu.Unlock()
		return err
	}
	return nil
}

func (service *Service) runWebSocketReader() {
	defer close(service.webSocketReaderDone)
	for {
		if service.webSockets.stopping() {
			return
		}
		event, err := service.runtime.TakeWebSocket(servicePollInterval)
		if err != nil {
			var runtimeErr *Error
			if errors.As(err, &runtimeErr) && runtimeErr.Code == ResultTimeout {
				continue
			}
			if service.webSockets.stopping() || errors.Is(err, errRuntimeClosed) ||
				(errors.As(err, &runtimeErr) && runtimeErr.Code == ResultClosed) {
				return
			}
			service.errors.add(err)
			return
		}
		service.webSockets.handle(*event)
	}
}

func (dispatcher *webSocketDispatcher) stopping() bool {
	dispatcher.mu.Lock()
	defer dispatcher.mu.Unlock()
	return dispatcher.closed
}

func (dispatcher *webSocketDispatcher) stopReader() {
	dispatcher.mu.Lock()
	dispatcher.closed = true
	dispatcher.mu.Unlock()
}

func (dispatcher *webSocketDispatcher) handle(event SocketEvent) {
	dispatcher.mu.Lock()
	if dispatcher.closed {
		dispatcher.mu.Unlock()
		return
	}
	if event.Kind == WebSocketAcceptFailed {
		delete(dispatcher.pending, event.OriginExchange)
		dispatcher.mu.Unlock()
		dispatcher.service.errors.add(errors.New("WebSocket upgrade failed"))
		return
	}
	if event.Kind == WebSocketOpen {
		plan := dispatcher.pending[event.OriginExchange]
		delete(dispatcher.pending, event.OriginExchange)
		if plan == nil || uint32(len(dispatcher.sessions)) >= dispatcher.capacity {
			dispatcher.mu.Unlock()
			_ = dispatcher.service.runtime.CloseWebSocket(event.Session, 1013, "upgrade owner unavailable")
			return
		}
		session := &WebSocketSession{
			runtime: dispatcher.service.runtime, id: event.Session,
			subprotocol: string(event.Data), open: true,
		}
		dispatcher.sessions[event.Session] = webSocketEntry{session: session, plan: plan}
		dispatcher.mu.Unlock()
		dispatcher.invoke(session, plan, event)
		return
	}
	entry, exists := dispatcher.sessions[event.Session]
	if event.Kind == WebSocketClose && exists {
		delete(dispatcher.sessions, event.Session)
		entry.session.mu.Lock()
		entry.session.open = false
		entry.session.mu.Unlock()
	}
	dispatcher.mu.Unlock()
	if exists {
		dispatcher.invoke(entry.session, entry.plan, event)
	}
}

func (dispatcher *webSocketDispatcher) invoke(session *WebSocketSession, plan *webSocketPlan, event SocketEvent) {
	defer func() {
		if recovered := recover(); recovered != nil {
			dispatcher.service.errors.add(fmt.Errorf("WebSocket handler panic: %v", recovered))
			_ = session.Close(1011, "handler failed")
		}
	}()
	if err := plan.handler(session, event); err != nil {
		dispatcher.service.errors.add(err)
		_ = session.Close(1011, "handler failed")
	}
}

func (dispatcher *webSocketDispatcher) shutdown() {
	dispatcher.mu.Lock()
	dispatcher.closed = true
	dispatcher.pending = nil
	for _, entry := range dispatcher.sessions {
		entry.session.mu.Lock()
		entry.session.open = false
		entry.session.mu.Unlock()
	}
	dispatcher.sessions = nil
	dispatcher.mu.Unlock()
}
