## ADDED Requirements

### Requirement: TLS flags

The proxy SHALL accept `--tls-cert` and `--tls-key` flags (and the env vars `TLS_CERT` and `TLS_KEY`), each naming a path to a PEM-encoded file. Both default to empty.

#### Scenario: Both flags set enable TLS

- **WHEN** the proxy is started with `--tls-cert=/path/to/cert.pem --tls-key=/path/to/key.pem`
- **THEN** the server listens on HTTPS using the provided certificate and key

#### Scenario: Neither flag set uses plain HTTP

- **WHEN** the proxy is started without either TLS flag
- **THEN** the server listens on plain HTTP

#### Scenario: Partial TLS config is rejected

- **WHEN** the proxy is started with `--tls-cert` set but no `--tls-key` (or vice versa)
- **THEN** the proxy exits with a non-zero status and an error stating that both must be set together
