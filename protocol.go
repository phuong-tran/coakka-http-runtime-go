package coakkahttp

import (
	"encoding/binary"
	"errors"
	"io"
	"strings"
	"unsafe"
)

const maximumHeaders = 100

// Four immutable entries fit a cheaper bounded scan. Larger collections keep
// the average O(1) index; this threshold is representation policy, not an HTTP
// or Core resource limit.
const smallHeaderScanLimit = 4

// Prepared buffered replies retain at most one ordinary Core-default header
// budget beside their immutable Go values. Larger valid declarations use the
// general staging path rather than growing an implicit response cache.
const maximumPreparedResponseHeaderBytes = 64 << 10

// Header positions and duplicate counts fit in one byte. A future increase
// of the public bound must deliberately widen the private index as well.
const _ uint8 = maximumHeaders

// ExchangeID is a generation-checked inbound exchange handle.
type ExchangeID struct {
	// Slot is the bounded Runtime slot.
	Slot uint64
	// Generation prevents stale handle reuse.
	Generation uint64
}

// WebSocketID is a generation-checked Runtime-owned session handle.
type WebSocketID struct {
	// Slot is the bounded Runtime slot.
	Slot uint64
	// Generation prevents stale handle reuse.
	Generation uint64
}

// OutboundID is a generation-checked Runtime-owned client call handle.
type OutboundID struct {
	// Slot is the bounded Runtime slot.
	Slot uint64
	// Generation prevents stale handle reuse.
	Generation uint64
}

// Header is one copied HTTP field. NewHeader validates it before native use.
type Header struct {
	// Name is an RFC token field name.
	Name string
	// Value excludes NUL, carriage return, and line feed.
	Value string
}

// NewHeader validates and returns one HTTP header.
func NewHeader(name, value string) (Header, error) {
	if name == "" || len(name) > maximumBridgeBytes {
		return Header{}, errors.New("HTTP header name is invalid")
	}
	for index := range len(name) {
		if !headerNameByte(name[index]) {
			return Header{}, errors.New("HTTP header name is invalid")
		}
	}
	if len(value) > maximumBridgeBytes || strings.ContainsAny(value, "\x00\r\n") {
		return Header{}, errors.New("HTTP header value is invalid")
	}
	return Header{Name: name, Value: value}, nil
}

func headerNameByte(value byte) bool {
	return value >= '0' && value <= '9' || value >= 'A' && value <= 'Z' ||
		value >= 'a' && value <= 'z' || strings.ContainsRune("!#$%&'*+-.^_`|~", rune(value))
}

// Headers is immutable and preserves declaration order and duplicate names.
// Its zero value is a usable empty collection with no backing allocation.
type Headers struct {
	entries []Header
	index   headerIndex
}

// All index state is built before publication, then shared read-only with the
// owned entries. next is allocated only for duplicate names; no lazy writes,
// synchronization or per-name value slice is needed during lookup.
type headerIndex struct {
	byName map[string]headerPositions
	next   *[maximumHeaders]uint8
}

type headerPositions struct {
	first uint8
	last  uint8
	count uint8
}

// NewHeaders validates and defensively copies every entry.
func NewHeaders(entries ...Header) (Headers, error) {
	if len(entries) > maximumHeaders {
		return Headers{}, errors.New("HTTP header count exceeds the Go bound")
	}
	if len(entries) == 0 {
		return Headers{}, nil
	}
	owned := make([]Header, len(entries))
	for index, entry := range entries {
		validated, err := NewHeader(entry.Name, entry.Value)
		if err != nil {
			return Headers{}, err
		}
		owned[index] = validated
	}
	return ownValidatedHeaders(owned), nil
}

// ownValidatedHeaders takes exclusive ownership of a bounded entry slice.
// Public callers validate/copy first; native callers have copied Core-validated
// fields under a live lease with boundary size/count checks. Never pass a
// caller-owned mutable slice here. No extra defensive copy or protocol scan is
// needed. Callers enforce len(owned) <= maximumHeaders before entering here.
// Construction is O(1) beyond owned storage for n <= 4; lookup is a bounded
// O(n) scan in that case. Larger collections build in average O(n) and use an
// average O(1), worst-case O(n) hash lookup, with n <= maximumHeaders.
// Normalization/hashing also reads name bytes on the indexed path.
func ownValidatedHeaders(owned []Header) Headers {
	if len(owned) == 0 {
		return Headers{}
	}
	if len(owned) <= smallHeaderScanLimit {
		return Headers{entries: owned}
	}
	index := headerIndex{byName: make(map[string]headerPositions, len(owned))}
	for position, entry := range owned {
		name := strings.ToLower(entry.Name)
		positions, found := index.byName[name]
		if !found {
			index.byName[name] = headerPositions{first: uint8(position), last: uint8(position), count: 1}
			continue
		}
		if index.next == nil {
			index.next = new([maximumHeaders]uint8)
		}
		// Append in O(1), even for interleaved duplicates. Links always advance
		// in entry order; count bounds traversal, so no sentinel is required.
		index.next[positions.last] = uint8(position)
		positions.last = uint8(position)
		positions.count++
		index.byName[name] = positions
	}
	return Headers{entries: owned, index: index}
}

// Get returns the first case-insensitive value for name in average O(1) time.
func (headers Headers) Get(name string) (string, bool) {
	if headers.index.byName == nil {
		for _, entry := range headers.entries {
			if strings.EqualFold(entry.Name, name) {
				return entry.Value, true
			}
		}
		return "", false
	}
	positions := headers.index.byName[strings.ToLower(name)]
	if positions.count == 0 {
		return "", false
	}
	return headers.entries[positions.first].Value, true
}

// GetAll returns every case-insensitive value for name in wire order using one
// average O(1) lookup plus O(k) traversal for k returned values. The returned
// slice is a defensive copy; unrelated entries are never scanned.
func (headers Headers) GetAll(name string) []string {
	if headers.index.byName == nil {
		count := 0
		for _, entry := range headers.entries {
			if strings.EqualFold(entry.Name, name) {
				count++
			}
		}
		if count == 0 {
			return nil
		}
		values := make([]string, 0, count)
		for _, entry := range headers.entries {
			if strings.EqualFold(entry.Name, name) {
				values = append(values, entry.Value)
			}
		}
		return values
	}
	positions := headers.index.byName[strings.ToLower(name)]
	if positions.count == 0 {
		return nil
	}
	values := make([]string, int(positions.count))
	position := positions.first
	values[0] = headers.entries[position].Value
	for i := 1; i < len(values); i++ {
		position = headers.index.next[position]
		values[i] = headers.entries[position].Value
	}
	return values
}

// Entries returns a defensive copy of the ordered fields.
func (headers Headers) Entries() []Header {
	return append([]Header(nil), headers.entries...)
}

// PathParameter is one named route capture with its encoded value.
type PathParameter struct {
	// Name is the copied route capture name.
	Name string
	// EncodedValue is the copied, still-encoded path value.
	EncodedValue string
}

// QueryParameter is one ordered encoded query item.
type QueryParameter struct {
	// EncodedKey is the copied, still-encoded key.
	EncodedKey string
	// EncodedValue is the copied, still-encoded value.
	EncodedValue string
	// HasValue distinguishes `key` from `key=`.
	HasValue bool
}

// Request is a fully Go-owned projection of an inbound request event.
type Request struct {
	// Exchange is the generation-checked exchange handle.
	Exchange ExchangeID
	// Method is the normalized HTTP method.
	Method string
	// Scheme is the request scheme reported by Runtime.
	Scheme string
	// Authority is the HTTP authority reported by Runtime.
	Authority string
	// Target is the encoded origin target including its query.
	Target string
	// Headers preserves request header order and duplicates.
	Headers Headers
	// PathParameters contains ordered route captures.
	PathParameters []PathParameter
	// QueryParameters contains ordered encoded query items.
	QueryParameters []QueryParameter
	// Trailers contains buffered request trailers when available on the request event.
	Trailers Headers
	// RouteID identifies the matched route.
	RouteID uint64
	// RouteGeneration identifies the matched route table generation.
	RouteGeneration uint64
	// BindingRevision identifies the handler binding revision.
	BindingRevision uint64
	// HandlerBindingID selects the application handler registered for this event.
	HandlerBindingID uint64
	// BodyDelivery reports inline or streamed body delivery.
	BodyDelivery BodyDelivery
	// Body reads the inline body or the streamed DATA events on a Service request.
	// Stream trailers become visible in Trailers after reads pass their event.
	Body io.ReadCloser
	body []byte
}

// Bytes returns a defensive copy of the inline request body.
func (request *Request) Bytes() []byte {
	if request == nil {
		return nil
	}
	return append([]byte(nil), request.body...)
}

// Text returns the inline request body as a Go string.
func (request *Request) Text() string {
	if request == nil {
		return ""
	}
	return string(request.body)
}

// Event is a copied inbound lane item. TakeEvent releases the native lease
// before returning, so every slice and nested value is Go-owned.
type Event struct {
	// Kind identifies the event payload.
	Kind EventKind
	// Exchange identifies the exchange.
	Exchange ExchangeID
	// Sequence is the Runtime event sequence.
	Sequence uint64
	// HandlerBindingID selects application dispatch.
	HandlerBindingID uint64
	// Data is copied event-specific stream data.
	Data []byte
	// Request is present for EventRequest.
	Request *Request
	// StreamTrailers is present for a streamed trailer event.
	StreamTrailers Headers
}

// Response is a copied buffered response or a declared streaming response.
type Response struct {
	// Status is an HTTP status from 100 through 599.
	Status uint32
	// Headers preserves response field order and duplicates.
	Headers   Headers
	body      []byte
	buffered  *bufferedResponsePlan
	producer  ResponseStreamProducer
	sse       SSEProducer
	webSocket *webSocketPlan
	file      *FileResponse
}

// bufferedResponsePlan is pointer-free wire storage owned by the Response.
// The private bridge borrows it for one synchronous call and constructs only
// stack views; Core still validates and copies accepted HTTP values.
type bufferedResponsePlan struct {
	wire        []byte
	headers     []Header
	bodyOffset  int
	headerCount uint32
}

func prepareBufferedResponse(headers Headers, body []byte) (*bufferedResponsePlan, []byte) {
	headerBytes := 0
	for _, entry := range headers.entries {
		headerBytes += len(entry.Name) + len(entry.Value)
		if headerBytes > maximumPreparedResponseHeaderBytes {
			return nil, append([]byte(nil), body...)
		}
	}
	tableBytes := len(headers.entries) * 2 * 4
	bodyOffset := tableBytes + headerBytes
	wire := make([]byte, bodyOffset+len(body))
	lengthOffset := 0
	byteOffset := tableBytes
	for _, entry := range headers.entries {
		binary.LittleEndian.PutUint32(wire[lengthOffset:], uint32(len(entry.Name)))
		binary.LittleEndian.PutUint32(wire[lengthOffset+4:], uint32(len(entry.Value)))
		lengthOffset += 8
		byteOffset += copy(wire[byteOffset:], entry.Name)
		byteOffset += copy(wire[byteOffset:], entry.Value)
	}
	copy(wire[bodyOffset:], body)
	plan := &bufferedResponsePlan{
		wire: wire, headers: headers.entries, bodyOffset: bodyOffset,
		headerCount: uint32(len(headers.entries)),
	}
	return plan, wire[bodyOffset:]
}

func (plan *bufferedResponsePlan) matches(headers Headers) bool {
	if plan == nil || len(plan.headers) != len(headers.entries) {
		return false
	}
	if len(plan.headers) == 0 {
		return true
	}
	return unsafe.SliceData(plan.headers) == unsafe.SliceData(headers.entries)
}

// ServeFile validates an application-selected file response under a configured
// Runtime authority. Runtime resolves and opens encodedPath after the handler returns.
func ServeFile(status uint32, headers Headers, authorityID uint64, encodedPath string) (Response, error) {
	head, err := NewResponse(status, headers, nil)
	if err != nil {
		return Response{}, err
	}
	if authorityID == 0 || encodedPath == "" || strings.ContainsRune(encodedPath, '\x00') {
		return Response{}, errors.New("file response authority or path is invalid")
	}
	head.file = &FileResponse{Head: head, AuthorityID: authorityID, EncodedPath: encodedPath}
	return head, nil
}

// NewResponse validates status/body bounds, shares immutable Headers and copies
// the caller's mutable body. Returned header slices remain defensive copies.
func NewResponse(status uint32, headers Headers, body []byte) (Response, error) {
	if status < 100 || status > 599 || len(body) > maximumBridgeBytes {
		return Response{}, errors.New("HTTP response status or body is invalid")
	}
	plan, ownedBody := prepareBufferedResponse(headers, body)
	return Response{Status: status, Headers: headers, body: ownedBody, buffered: plan}, nil
}

// Bytes returns a defensive copy of the response body.
func (response Response) Bytes() []byte {
	return append([]byte(nil), response.body...)
}

// Text returns a validated UTF-8-compatible plain text response.
func Text(status uint32, body string) (Response, error) {
	header, err := NewHeader("content-type", "text/plain; charset=utf-8")
	if err != nil {
		return Response{}, err
	}
	headers, err := NewHeaders(header)
	if err != nil {
		return Response{}, err
	}
	return NewResponse(status, headers, []byte(body))
}

// Bytes returns a validated response that owns a copy of body.
func Bytes(status uint32, body []byte) (Response, error) {
	return NewResponse(status, Headers{}, body)
}

// Empty returns a validated response with no headers or body.
func Empty(status uint32) (Response, error) {
	return NewResponse(status, Headers{}, nil)
}

// NewStreamingResponse validates and copies a response head. The producer runs
// on the fixed Service worker after Runtime accepts the head. It writes only under
// Runtime writable credit and returns the copied trailers for the final frame.
func NewStreamingResponse(status uint32, headers Headers, producer ResponseStreamProducer) (Response, error) {
	if producer == nil {
		return Response{}, errors.New("response stream producer is nil")
	}
	response, err := NewResponse(status, headers, nil)
	if err != nil {
		return Response{}, err
	}
	response.producer = producer
	return response, nil
}

// NewSSEResponse validates and copies extra response headers. The producer
// writes typed events through one bounded Service worker and Runtime's stream
// pressure lane. Runtime supplies status 200 and the SSE protocol headers.
func NewSSEResponse(headers Headers, producer SSEProducer) (Response, error) {
	if producer == nil {
		return Response{}, errors.New("SSE producer is nil")
	}
	response, err := NewResponse(200, headers, nil)
	if err != nil {
		return Response{}, err
	}
	response.sse = producer
	return response, nil
}

// FileResponse serves a path under a configured Runtime file authority.
type FileResponse struct {
	// Head supplies status and headers; its body should be empty.
	Head Response
	// AuthorityID selects a configured file root.
	AuthorityID uint64
	// EncodedPath is resolved and opened by Runtime under that root.
	EncodedPath string
}

// Failure submits a typed application failure as the exchange's first completion.
type Failure struct {
	// Reason is the Runtime-defined failure reason.
	Reason uint32
	// Detail is copied into Runtime's bounded diagnostic storage.
	Detail string
}

// SSEEvent is one copied server-sent event write.
type SSEEvent struct {
	// Data is the event payload.
	Data []byte
	// EventType is the optional event field.
	EventType string
	// ID is the optional id field.
	ID string
	// RetryMillis is the optional client retry delay.
	RetryMillis uint64
	// HasEventType distinguishes an empty field from absence.
	HasEventType bool
	// HasID distinguishes an empty field from absence.
	HasID bool
	// HasRetry distinguishes zero from absence.
	HasRetry bool
}

// SocketEvent is a copied WebSocket lane item. TakeWebSocket releases its
// native lease before returning.
type SocketEvent struct {
	// Kind identifies the WebSocket event.
	Kind SocketEventKind
	// Session is the generation-checked WebSocket handle.
	Session WebSocketID
	// OriginExchange identifies the upgrade request.
	OriginExchange ExchangeID
	// CloseCode carries a close event's status code.
	CloseCode uint32
	// Data is copied event data or close reason.
	Data []byte
}

// ClientRequest submits one buffered outbound call to a logical target.
type ClientRequest struct {
	// TimeoutMillis is the monotonic call deadline.
	TimeoutMillis uint32
	// LogicalTarget selects a configured target.
	LogicalTarget string
	// Method is the outbound HTTP method.
	Method string
	// Target is the encoded origin target.
	Target string
	// Headers preserves outbound field order and duplicates.
	Headers Headers
	// Body is copied before SubmitOutbound returns successfully.
	Body []byte
}

// ClientTerminal is a copied outbound terminal. TakeOutbound releases the
// native response lease before returning.
type ClientTerminal struct {
	// Reason is Runtime's terminal reason.
	Reason OutboundReason
	// Call is the generation-checked outbound call handle.
	Call OutboundID
	// LogicalTarget is the selected logical target.
	LogicalTarget string
	// TargetGeneration is the selected topology generation.
	TargetGeneration uint64
	// SelectedNodeID is the selected endpoint node.
	SelectedNodeID string
	// Phase is Runtime's terminal phase.
	Phase OutboundPhase
	// Retry is Runtime's retry disposition.
	Retry OutboundRetry
	// Certainty is Runtime's delivery certainty.
	Certainty OutboundCertainty
	// ResponseStatus is the received HTTP status.
	ResponseStatus uint16
	// ProviderCode is the native provider result.
	ProviderCode int32
	// ResponseHeaders are copied before lease release.
	ResponseHeaders Headers
	// ResponseBody is copied before lease release.
	ResponseBody []byte
	// Diagnostic is copied before lease release.
	Diagnostic string
}

// Handler processes one copied inline Request and returns one buffered response.
type Handler func(*Request) (Response, error)
