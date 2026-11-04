# CoAkka HTTP Runtime for Go

![CoAkka](https://raw.githubusercontent.com/phuong-tran/coakka-samples/a539dfcd7dc8d93c5b1c61b1324899af13d47ca6/docs/common/assets/brand/coakka-logo.png)

Use CoAkka HTTP Runtime to serve HTTP requests, stream responses, handle
WebSockets, and call configured upstream services from a Go application.

## Requirements

Use Go 1.23 or newer, cgo, and a C11 compiler for your target. Run
`go env GOOS GOARCH` to choose the SDK for the application you will build.
For WSL2, select the Linux row for the architecture reported by Go.

| `GOOS/GOARCH` | Platform | SDK archive suffix |
| --- | --- | --- |
| `darwin/arm64` | macOS 11 or newer | `macos-aarch64` |
| `linux/amd64` | Linux x86-64, glibc 2.28 or newer | `linux-x86_64` |
| `linux/arm64` | Linux ARM64, glibc 2.28 or newer | `linux-aarch64` |
| `windows/amd64` | Windows x64 | `windows-x86_64` |
| `windows/arm64` | Windows ARM64 | `windows-aarch64` |

On Windows, put a cgo-compatible compiler for the selected architecture on
`PATH` or point `CC` to it. Use a directory without spaces for the extracted SDK.

## Install v2.0.1

Download `coakka-http-runtime-native-2.0.1-<suffix>.tar.gz` and
`SHA256SUMS` from the [v2.0.1 GitHub Release](https://github.com/phuong-tran/coakka-http-runtime-go/releases/tag/v2.0.1).
Put both files in your application directory. Compare the archive's hash with
its line in `SHA256SUMS` before extracting it. On macOS use
`shasum -a 256 <archive>`, on Linux use `sha256sum <archive>`, and on Windows
use `Get-FileHash <archive> -Algorithm SHA256`.

Run `go mod init example.com/coakka-hello` if you are starting a new module.
From that directory, fetch the Go connector.

```sh
go get github.com/phuong-tran/coakka-http-runtime-go/v2@v2.0.1
```

The Go tag supplies the connector source. The matching SDK archive supplies
the headers and shared library for cgo. Keep both on version 2.0.1.

On macOS or Linux, extract the archive and set these variables in the terminal
where you will build and run the examples. The filename below is for macOS
ARM64. Use the filename for your target from the table.

```sh
mkdir -p sdk
tar -xzf coakka-http-runtime-native-2.0.1-macos-aarch64.tar.gz -C sdk
export COAKKA_HTTP_SDK="$PWD/sdk"
export CGO_ENABLED=1
export CGO_CFLAGS="-I$COAKKA_HTTP_SDK/include"
export CGO_LDFLAGS="-L$COAKKA_HTTP_SDK/lib -Wl,-rpath,$COAKKA_HTTP_SDK/lib"
```

On Windows, use PowerShell. Substitute the ARM64 archive if you are building
for Windows ARM64.

```powershell
New-Item -ItemType Directory -Force sdk | Out-Null
tar.exe -xzf coakka-http-runtime-native-2.0.1-windows-x86_64.tar.gz -C sdk
$env:COAKKA_HTTP_SDK = (Resolve-Path sdk).Path.Replace('\', '/')
$env:CGO_ENABLED = '1'
$env:CGO_CFLAGS = "-I$env:COAKKA_HTTP_SDK/include"
$env:CGO_LDFLAGS = "-L$env:COAKKA_HTTP_SDK/lib"
$env:PATH = "$env:COAKKA_HTTP_SDK/bin;$env:PATH"
```

If the build cannot find `coakka/http/host.h`, check `CGO_CFLAGS` and confirm the
SDK directory contains `include/` and `lib/`. If the program fails to load the
shared library, check that the SDK matches the binary's architecture. On
macOS and Linux, keep the versioned library links when deploying and choose a
loader path for the deployed application. On Windows, put the matching DLL
beside the deployed executable.

On macOS, if the linker reports `unknown architecture` in a system `.tbd`
file, use a macOS SDK that matches your compiler and linker. With full Xcode
selected, set the SDK for this terminal before retrying the build.

```sh
export SDKROOT="$(xcodebuild -version -sdk macosx Path)"
```

## First request

Save this example as `main.go`. It listens on loopback and asks the runtime
to choose an available port.

```go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/phuong-tran/coakka-http-runtime-go/v2"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := serve(ctx); err != nil {
		log.Fatal(err)
	}
}

func serve(ctx context.Context) (err error) {
	service, err := coakkahttp.NewBuilder().
		Listen("127.0.0.1", 0).
		Get("/hello/{name}", func(request *coakkahttp.Request) (coakkahttp.Response, error) {
			name := request.PathParameters[0].EncodedValue
			return coakkahttp.Text(http.StatusOK, "hello "+name)
		}).
		Start()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, service.Close())
	}()

	port, err := service.Port()
	if err != nil {
		return err
	}
	log.Printf("listening on http://127.0.0.1:%d", port)
	<-ctx.Done()
	return nil
}
```

Run `go run .`. From a second terminal, open `/hello/Ada` at the printed
address. For example, if the server prints port `49321`, run
`curl -i http://127.0.0.1:49321/hello/Ada`. In Windows PowerShell, use
`curl.exe`. The response is HTTP 200 with the body `hello Ada`.
Press Ctrl+C in the server terminal to stop it.

## Outbound requests

With the server running, save this example as `client/main.go` in the same
module. In another terminal with the same SDK environment, run
`go run ./client 49321` using the port printed by your server.
The client prints `200 hello Ada`.

```go
package main

import (
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/phuong-tran/coakka-http-runtime-go/v2"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: go run ./client <server-port>")
	}
	port, err := strconv.ParseUint(os.Args[1], 10, 16)
	if err != nil || port == 0 {
		log.Fatalf("invalid server port %q", os.Args[1])
	}
	if err := call(uint16(port)); err != nil {
		log.Fatal(err)
	}
}

func call(port uint16) (err error) {
	const host = "127.0.0.1"
	target := coakkahttp.OutboundTarget{
		Name:       "hello",
		Generation: 1,
		Endpoints: []coakkahttp.OutboundEndpoint{{
			NodeID:        "local",
			ConnectHost:   host,
			ConnectPort:   port,
			HTTPAuthority: net.JoinHostPort(host, strconv.Itoa(int(port))),
		}},
	}
	service, err := coakkahttp.NewBuilder().OutboundTarget(target).Start()
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, service.Close())
	}()

	callID, err := service.SubmitOutbound(coakkahttp.ClientRequest{
		LogicalTarget: target.Name,
		Method:        http.MethodGet,
		Target:        "/hello/Ada",
		TimeoutMillis: 3000,
	})
	if err != nil {
		return err
	}
	result, err := service.TakeOutbound(5 * time.Second)
	if err != nil {
		return err
	}
	if result == nil {
		return errors.New("no outbound result within five seconds")
	}
	if result.Call != callID || result.Reason != coakkahttp.OutboundResponse {
		return fmt.Errorf("unexpected outbound result for %v: %s", result.Call, result.Reason)
	}
	fmt.Printf("%d %s\n", result.ResponseStatus, result.ResponseBody)
	return nil
}
```

`ConnectHost` and `ConnectPort` choose the connection destination.
`HTTPAuthority` sets the HTTP authority, and TLS uses its configured peer
identity. A completed HTTP response can carry a 4xx or 5xx status. Inspect
`Reason` to distinguish it from a transport or lifecycle failure.

## Upgrade from v1

1. Change imports of `github.com/phuong-tran/coakka-http-runtime-go` to
   `github.com/phuong-tran/coakka-http-runtime-go/v2`. The package name remains
   `coakkahttp`. Update dependencies that still import the v1 connector.
2. Update the application to use the v2.0.1 Go module and the matching 2.0.1
   native SDK. Replace the build and runtime library paths together.
3. Run `go mod tidy` and `go test ./...` in your application.

## API reference

Run `go doc github.com/phuong-tran/coakka-http-runtime-go/v2` in an application
that has fetched the module. Use `Builder` for the Go service API or `Runtime`
for the lower-level pull API.

- [URL grammar and glossary](https://github.com/phuong-tran/coakka-samples/blob/main/docs/coakka-http-runtime/reference/glossary.md)
- [Go v1.1.2 sample and feature recipes](https://github.com/phuong-tran/coakka-samples/blob/7fab741dec51b2d8c156f653cdbe8442c56f2704/docs/coakka-http-runtime/go/integration.md)

## Response compression

Configure buffered responses with
`builder.Compression(&coakkahttp.Compression{Mode: coakkahttp.CompressionGZIP, MinimumBodyBytes: 1, GZIPLevel: 6})`
before startup. The builder copies the optional value; `Compression(nil)`
removes an earlier override. Explicit disable uses
`&coakkahttp.Compression{Mode: coakkahttp.CompressionDisabled}` without GZIP-only
tuning fields. The runtime rejects incompatible settings rather than ignoring them.

The runtime owns response eligibility, resource ceilings and negotiation. Handlers
return normal application values; do not add another compression layer. The
Go boundary rejects an unknown mode that cannot be represented by the native
boolean. Numeric bounds and capability decisions remain runtime-owned. This is
startup configuration, not a live change to existing connections.

The transform applies to eligible buffered responses, not response streams or
SSE. Streams remain uncompressed and bounded; an identity-refusing client
receives HTTP 406 before an unencoded stream starts when this policy is enabled.
Application-supplied content encoding remains application-owned.

## I/O backend selection

The default configuration leaves `Config.IOBackend` at zero, which selects the
platform backend. On Linux this is the normal epoll path. To request io_uring:

```go
runtime, err := coakkahttp.Open(coakkahttp.Config{
	IOBackend: coakkahttp.IOUring,
	Routes: []coakkahttp.Route{{
		ID: 1, HandlerBindingID: 1, Method: "GET", Pattern: "/ready",
	}},
})
```

The connector does not inspect the kernel or issue a probe. The native library
owns eligibility, initialization, and fallback. After `Start`, read
`RuntimeInfo()` to distinguish requested and effective backends and to inspect
the typed fallback reason.

## Supported service features

The current Go surface includes:

- HTTP/1.1, HTTP/2, and HTTP/3 listener declarations
- plaintext, TLS, and mutual TLS
- buffered and streamed request bodies, including trailers
- buffered responses, response streams, trailers, and server-sent events
- WebSocket upgrade, receive, send, writable, close, and failure events
- gzip response compression with native defaults for omitted bounds
- immutable static trees, directory index, SPA fallback, and application file
  authorities
- logical outbound targets, TLS trust and client identities, submit, cancel,
  and terminal reads
- health snapshots and bounded active liveness probes
- disabled-by-default monitoring with fixed reservations and live policy apply
- handler rebinding and complete route-table publication with generation and
  revision checks

All counts, queues, bodies, retained bytes, deadlines, and shutdown waits have
construction-time bounds. Zero-valued optional limits select bounded native
defaults.

## Low-level example

```go
runtime, err := coakkahttp.Open(coakkahttp.Config{
	Listeners: []coakkahttp.Listener{{
		ID:          1,
		BindAddress: "127.0.0.1",
		Protocol:    coakkahttp.ProtocolHTTP11,
		Security:    coakkahttp.SecurityPlaintext,
	}},
	Routes: []coakkahttp.Route{{
		ID:               1,
		HandlerBindingID: 1,
		Method:           "POST",
		Pattern:          "/echo",
	}},
})
if err != nil {
	return err
}
defer runtime.Close()

if err = runtime.Start(); err != nil {
	return err
}
event, err := runtime.TakeEvent(250 * time.Millisecond)
if err != nil {
	return err
}
if event.Kind == coakkahttp.EventRequest {
	reply, _ := coakkahttp.Bytes(200, event.Request.Bytes())
	err = runtime.Respond(event.Exchange, reply)
}
```

Use exactly one goroutine for each pull lane. Returned events are Go-owned
copies: the connector releases the native lease before the method returns.

## Monitoring

Monitoring starts disabled. Reserve the resources that may be enabled later:

```go
builder.Monitor(coakkahttp.MonitorOptions{
	Collection:       coakkahttp.MonitorDisabled,
	EventCapacity:    256,
	MaxEventsPerRead: 32,
	AggregateCategories: coakkahttp.MonitorCategoryLifecycle |
		coakkahttp.MonitorCategoryExchange,
	EventCategories: coakkahttp.MonitorCategoryLifecycle,
	SignalReserved:  true,
})
```

`MonitorConfig`, `MonitorSnapshot`, generation-checked `ApplyMonitorPolicy`,
bounded event reads, wait, and interrupt all delegate to the native owner.
Monitor saturation never changes an HTTP exchange result.

## Route changes

`Service.Routes()` and `Runtime.Routes()` return `(RouteSnapshot, error)` from
the runtime: one complete structural generation, binding-change sequence and bounded
route/binding list. The caller owns the copy; mutation cannot change routing.
Monitoring need not be enabled. Refusal or unresolved control returns no partial
snapshot.

Prepare a new handler binding before publishing it. `RebindHandler` changes only
the selected route's binding revision; `PublishRoutes` replaces one complete
immutable structural generation. Admitted requests continue through the
binding they captured. Rejected changes preserve the previous effective state.

Application dispatch uses bounded maps keyed by handler identity. The connector
does not linearly scan handlers on the request path; structural matching and
route identity lookup remain native-owned.

## Shutdown

Outbound terminals preserve typed `OutboundReason`, `OutboundPhase`,
`OutboundRetry` and `OutboundCertainty`, including unknown numbers. Use named
constants: HTTP 404 or 500 can be `OutboundResponse`, while cancellation and deadline
expiry remain distinct causes. `Reason.String()` derives a code without storing
a duplicate name. Retry disposition does not authorize replaying business work;
the application owns idempotency and retry budgets. Never expose operator
diagnostics automatically in an HTTP response.

`Service.Close` stops admission, observes drain outcome and joins Go work before
destruction. Application handlers must return. Inspect
`*CloseError.RuntimeRetained` after refusal and retain the service when cleanup
needs retry. `*DrainError` preserves forced expiry or abort even after cleanup.
Low-level `Runtime.Close` is forced cleanup: graceful low-level use requires
`Drain`, progressing readers and handlers, then `WaitDrained` first. A timed wait
is not proof of completion; an expired drain is not graceful success.

[HTTP Runtime documentation](https://github.com/phuong-tran/coakka-samples/tree/main/docs/coakka-http-runtime)
