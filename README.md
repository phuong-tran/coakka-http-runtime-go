# CoAkka HTTP Runtime for Go

![CoAkka](https://raw.githubusercontent.com/phuong-tran/coakka-samples/a539dfcd7dc8d93c5b1c61b1324899af13d47ca6/docs/common/assets/brand/coakka-logo.png)

**New application?** Start with the [Go integration guide](https://github.com/phuong-tran/coakka-samples/blob/main/coakka-http-runtime/go/integration.md)
for installation, a first request and feature-by-feature usage. This page
remains the package/API reference.

`coakkahttp` lets a Go app serve HTTP and WebSocket requests and make outbound
HTTP calls through CoAkka HTTP Runtime. Install the Go module and the matching
native SDK before building the app.

## Contents

- [URL grammar and glossary](https://github.com/phuong-tran/coakka-publish/blob/main/coakka-http-runtime/docs/glossary.md)

- [Requirements](#requirements)
- [Install 2.0.1](#install-201)
- [Quick start](#quick-start)
- [I/O backend selection](#io-backend-selection)
- [Supported service features](#supported-service-features)
- [Response compression](#response-compression)
- [Low-level example](#low-level-example)
- [Monitoring](#monitoring)
- [Route changes](#route-changes)
- [Shutdown](#shutdown)

## Requirements

- Go 1.23 or newer and a C11 compiler that cgo can use for your target.
- CoAkka HTTP native SDK 2.0.1 for the app's operating system and architecture.

The SDK targets Linux x86-64 and ARM64 with glibc 2.28 or newer, macOS ARM64
with macOS 11 or newer, and Windows x64 and ARM64. On WSL2, use the Linux
SDK for the architecture reported by Go. Check the target with
`go env GOOS GOARCH`.

## Install 2.0.1

Download the SDK for your target and its checksum from the
[Go connector GitHub release](https://github.com/phuong-tran/coakka-http-runtime-go/releases/tag/v2.0.1).
Verify the archive checksum before extracting it. Use an absolute SDK path
with no spaces for the commands below.

Compare the archive's hash with the line for that filename in `SHA256SUMS`.
On Linux use `sha256sum <archive>`, on macOS use `shasum -a 256 <archive>`,
and on Windows use `Get-FileHash <archive> -Algorithm SHA256`.

From your app's module directory, fetch the tagged Go module. For a new app,
create a directory and run `go mod init example.com/hello` first.

```sh
go get github.com/phuong-tran/coakka-http-runtime-go/v2@v2.0.1
```

On macOS or Linux, set `COAKKA_HTTP_SDK` to the extracted directory that
contains `include/` and `lib/`.

```sh
export COAKKA_HTTP_SDK=/absolute/path/to/sdk
export CGO_ENABLED=1
export CGO_CFLAGS="-I$COAKKA_HTTP_SDK/include"
export CGO_LDFLAGS="-L$COAKKA_HTTP_SDK/lib -Wl,-rpath,$COAKKA_HTTP_SDK/lib"
```

On Windows, run the equivalent setup in PowerShell. Use a cgo-compatible
compiler for your target architecture and set `CC` if it is outside your
`PATH`.

```powershell
$env:COAKKA_HTTP_SDK = 'C:/coakka/http-sdk-2.0.1'
$env:CGO_ENABLED = '1'
$env:CGO_CFLAGS = "-I$env:COAKKA_HTTP_SDK/include"
$env:CGO_LDFLAGS = "-L$env:COAKKA_HTTP_SDK/lib"
$env:PATH = "$env:COAKKA_HTTP_SDK/bin;$env:PATH"
```

For deployment, keep the matching library with your application. Preserve
the library aliases on macOS and Linux, or place the Windows DLL beside
the executable. The macOS and Linux example above uses an absolute loader
path for local development. Choose the library location for the deployed
application when building its binary.

## Quick start

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"

    coakkahttp "github.com/phuong-tran/coakka-http-runtime-go/v2"
)

func main() {
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
    defer stop()
    service, err := coakkahttp.NewBuilder().
        Get("/hello/{name}", func(request *coakkahttp.Request) (coakkahttp.Response, error) {
            name := request.PathParameters[0].EncodedValue
            return coakkahttp.Text(200, "hello "+name)
        }).
        Start()
    if err != nil {
        log.Fatal(err)
    }
    port, err := service.Port()
    if err != nil {
        closeErr := service.Close()
        log.Fatalf("read port: %v; close: %v", err, closeErr)
    }
    log.Printf("listening on http://127.0.0.1:%d", port)
    <-ctx.Done()
    if err := service.Close(); err != nil {
        log.Fatal(err)
    }
}
```

`Builder` supplies a bounded request/reply facade. `Runtime` is the lower-level
pull API for applications that need direct ownership of events and commands.
Both use the same host API and the same native service.

Save the example as `main.go` and run `go run .`. Copy the port from the
`listening on` message, then request `/hello/Ada` at that address with
`curl` or a browser. The response is HTTP 200 with the body `hello Ada`.
Press Ctrl+C to close the service.

When upgrading from 1.x, change Go imports to
`github.com/phuong-tran/coakka-http-runtime-go/v2` and update the SDK
directory to 2.0.1. Run `go mod tidy` and rebuild your app. If compilation cannot find `coakka/http/host.h`,
check `CGO_CFLAGS`. If the executable cannot load the shared library, check
the SDK architecture and the library location used at runtime.

## Response compression

Configure ordinary host-inlined handlers with
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
receives HTTP406 before an unencoded stream starts when this policy is enabled.
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
    Collection:          coakkahttp.MonitorDisabled,
    EventCapacity:       256,
    MaxEventsPerRead:    32,
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
snapshot. This unreleased contract replaces the old local-declaration getter.

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
constants: HTTP404/500 can be `OutboundResponse`, while cancellation and deadline
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
