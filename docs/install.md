---
icon: lucide/package
---

# Install

## Build from source

Requires Go **1.25** or later (see `go.mod`).

```sh
git clone https://github.com/gp42/aws-outbound-jwt-proxy
cd aws-outbound-jwt-proxy
make build
```

The binary is written to `bin/aws-outbound-jwt-proxy`.

For cross-platform release builds covering Linux, macOS, and Windows on amd64/arm64:

```sh
make build-all
```

Outputs land in `dist/aws-outbound-jwt-proxy-<os>-<arch>[.exe]`.

## Run directly with `go run`

Useful for local hacking:

```sh
go run .
```

## Verify the binary

```sh
./bin/aws-outbound-jwt-proxy --help
```

You should see the flag list. See [Configuration](configuration.md) for the full reference, or jump straight to [Quick start](quick-start.md) to try it end-to-end.
