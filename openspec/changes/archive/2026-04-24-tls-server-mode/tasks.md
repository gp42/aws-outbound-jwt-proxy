## 1. Configuration

- [x] 1.1 Add `TLSCert string` and `TLSKey string` fields to `config.Config`
- [x] 1.2 Register `--tls-cert` and `--tls-key` flags in `BindFlags` (default empty, path to PEM file)
- [x] 1.3 Add validation: `TLSCert` and `TLSKey` must both be empty or both be non-empty; otherwise return an error
- [x] 1.4 Unit tests: neither set, both set, only cert, only key

## 2. Server

- [x] 2.1 In `server.New`, if both TLS fields are set, configure the `*http.Server` so the caller uses `ListenAndServeTLS`
- [x] 2.2 Add a method or package-level helper so `cmd/root.go` can decide between `ListenAndServe` and `ListenAndServeTLS(cert, key)` based on config
- [x] 2.3 `cmd/root.go` calls the correct listen method; logs include whether TLS is enabled

## 3. Verify

- [x] 3.1 `go build ./...` and `go test ./...` pass
- [x] 3.2 Generate a self-signed cert/key with `openssl`; start proxy with both flags; curl `--insecure https://...` returns `204`
- [x] 3.3 Start with only `--tls-cert`; proxy exits with the both-or-neither error
- [x] 3.4 Start without TLS flags; plain HTTP still works
