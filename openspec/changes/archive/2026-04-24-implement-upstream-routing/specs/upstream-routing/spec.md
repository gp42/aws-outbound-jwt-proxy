## ADDED Requirements

### Requirement: Pinned upstream takes precedence

When the proxy is configured with a pinned upstream host, the resolver SHALL use the pinned host and scheme for every request and SHALL NOT consult any request header for routing.

#### Scenario: Pinned host with header present

- **WHEN** the proxy is started with `--upstream-host=api.example.com` and `--upstream-scheme=https`, and a request arrives carrying `X-Upstream-Host: other.example.com`
- **THEN** the resolver returns a target URL with host `api.example.com` and scheme `https`, ignoring the header

#### Scenario: Pinned host with no header

- **WHEN** the proxy is started with `--upstream-host=api.example.com`, and a request arrives with no upstream header
- **THEN** the resolver returns a target URL with host `api.example.com`

### Requirement: Header-based upstream when unpinned

When no pinned upstream is configured, the resolver SHALL read the target host from the request header named by `--host-header` (default `X-Upstream-Host`) and use the configured `--upstream-scheme` as the target scheme.

#### Scenario: Default header carries upstream

- **WHEN** the proxy is started without `--upstream-host`, and a request arrives with `X-Upstream-Host: api.example.com`
- **THEN** the resolver returns a target URL with host `api.example.com`

#### Scenario: Custom header name

- **WHEN** the proxy is started with `--host-header=X-Target` and no pinned upstream, and a request arrives with `X-Target: api.example.com`
- **THEN** the resolver returns a target URL with host `api.example.com`

#### Scenario: Host header as upstream source

- **WHEN** the proxy is started with `--host-header=Host` and no pinned upstream, and a request arrives whose HTTP `Host` header is `api.example.com`
- **THEN** the resolver returns a target URL with host `api.example.com`, reading from the request's Host line rather than from `Header`

### Requirement: Target URL preserves path and query

The resolver SHALL preserve the inbound request's path and raw query string in the resolved target URL.

#### Scenario: Path and query preserved

- **WHEN** a request arrives for `/v1/items?limit=10` and resolves to host `api.example.com` with scheme `https`
- **THEN** the resolved target URL is `https://api.example.com/v1/items?limit=10`

### Requirement: Unresolvable upstream is rejected

When no upstream can be resolved (no pinned host and the configured header is missing or empty), the proxy SHALL respond with HTTP `400 Bad Request` and SHALL NOT attempt to forward the request.

#### Scenario: No pin, no header

- **WHEN** the proxy is started without `--upstream-host`, and a request arrives with no value in the configured header
- **THEN** the proxy responds `400 Bad Request` with a body naming the expected header

### Requirement: Stub forwarding behavior

Until the request-forwarding capability is implemented, after resolving a target the proxy SHALL log the resolved target to stdout and respond with HTTP `204 No Content`.

#### Scenario: Target resolved

- **WHEN** a request is successfully resolved to `https://api.example.com/foo`
- **THEN** the proxy writes a line containing `https://api.example.com/foo` to stdout and responds `204 No Content`
