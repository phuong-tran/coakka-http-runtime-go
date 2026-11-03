# CoAkka HTTP Runtime for Go

<p align="center">
  <img src="https://raw.githubusercontent.com/phuong-tran/coakka-samples/a539dfcd7dc8d93c5b1c61b1324899af13d47ca6/docs/common/assets/brand/coakka-logo.png" alt="CoAkka" width="480">
</p>

**One HTTP runtime, with idiomatic Go services, values and errors.**

CoAkka HTTP Runtime is part of the CoAkka ecosystem, alongside CoAkka Runtime
and CoAkka Logger. It supplies a shared HTTP server and client for applications
written in different languages. Business handlers stay in the application; the
HTTP runtime owns protocol I/O, routing, bounded resources and lifecycle.

Use it for HTTP APIs, outbound HTTP requests, static files, streamed responses,
server-sent events and WebSocket sessions. Each language keeps its own API,
values and scheduling model.

## Install and build

```sh
go get github.com/phuong-tran/coakka-http-runtime-go@v1.0.0
```

Import `github.com/phuong-tran/coakka-http-runtime-go` as `coakkahttp`. Go1.23
or newer, a C11 cgo compiler and the matching CoAkka HTTP native SDK are
required. This module contains connector source; the SDK is installed
separately and carries its own license and notices.

For a non-system SDK installation:

```sh
CGO_ENABLED=1 \
CGO_CFLAGS="-I$COAKKA_HTTP_PREFIX/include" \
CGO_LDFLAGS="-L$COAKKA_HTTP_PREFIX/lib" \
go build ./...
```

Make the SDK library directory available to the operating-system loader.
Linux and macOS builds may set an explicit installation rpath in
`CGO_LDFLAGS`; on Windows, place the SDK `bin` directory on `PATH`. Do not
mix a connector release with native libraries from another release.

## Platforms

The package family covers macOS ARM64, Linux ARM64 and x86_64, and Windows
ARM64 and x86_64. WSL2 uses the Linux x86_64 package. Linux requires glibc2.28
or newer; macOS requires version11 or newer. Qualification and registry
availability belong to the exact release record.

## Documentation and examples

- [HTTP Runtime documentation](https://github.com/phuong-tran/coakka-samples/tree/main/docs/coakka-http-runtime)
- [Runnable HTTP examples](https://github.com/phuong-tran/coakka-samples/tree/main/coakka-http-runtime)
- [Artifacts, checksums and release records](https://github.com/phuong-tran/coakka-publish/tree/main/coakka-http-runtime)
- [CoAkka ecosystem](https://github.com/phuong-tran/coakka-samples/tree/main/docs/common)
- [Questions and issues](https://github.com/phuong-tran/coakka-samples/issues)

Licensing scopes and required notices are described in `PACKAGE-LICENSE.md`.
