# CoAkka HTTP Runtime for Go changelog

## 2.3.1

Core generation **2.3**, native SDK **2.3.0**, C ABI **10**. The final `1`
is the first Go connector revision on this Core generation.

- Add `MultiProtocolListener` on the builder and `OpenMultiProtocol` on
  Runtime. One TLS endpoint admits HTTP/1.1, HTTP/2 and HTTP/3 on the same
  port. `MultiProtocolListenerInfo` reports the endpoint accepted by Core.
- Keep the shared listener's buffered request/response scope explicit. Use
  standalone listeners for upload/response streaming, SSE, WebSocket,
  mTLS, live route publication and live TLS credential changes.
- Preserve streamed request credit until the Service handler consumes or
  discards DATA and TRAILERS. The Go dispatcher returns each Core lease
  exactly once, including cancellation and close.
- Adopt the qualified Core corrections for HTTP/2 upload recovery and
  request lifetime, HTTP/3 stream credit, response streaming/SSE, drain,
  timeout wakeups and Windows UDP/timer behavior. Standalone HTTP/2 and
  HTTP/3 WebSocket use Extended CONNECT; the native ABI remains 10.
- Refresh installation, the first request and a complete same-port
  MultiProtocol example. Add ecosystem, benchmark and support links.
  GitHub Sponsors and Ko-fi are configured in `.github/FUNDING.yml`.

Go **1.23** remains the minimum. Install the module tag `v2.3.1` and its
matching SDK 2.3.0 archive from the GitHub Release. The five native targets
remain Linux ARM64/x86-64, macOS ARM64 and Windows ARM64/x64. WSL2 uses the
Linux x86-64 artifact.

## 2.2.1

First Go connector revision on Core generation 2.2, using SDK 2.2.0 and
ABI 10. Adopt the standalone HTTP/3 wakeup corrections. Preserve the
existing standalone listener APIs and Go 1.23 minimum.

[Release history](https://github.com/phuong-tran/coakka-http-runtime-go/releases) ·
[GitHub Sponsors](https://github.com/sponsors/phuong-tran) ·
[Ko-fi](https://ko-fi.com/phuongnamtran)
