## ADDED Requirements

### Requirement: Host-derived audience resolver

The proxy SHALL provide a `token.AudienceResolver` implementation, `HostAudience`, that derives the audience set for an outbound request from the target URL fields (`req.URL.Scheme` and `req.URL.Host`) written by the router. The resolver SHALL return a single-element slice whose value is of the form `<scheme>://<host>`, with scheme lowercased, host lowercased, and any port stripped from the host.

#### Scenario: Host with explicit port is normalized

- **WHEN** the forwarder invokes `HostAudience.Resolve` on a request whose `URL.Scheme` is `https` and `URL.Host` is `API.example.com:443`
- **THEN** the resolver returns `["https://api.example.com"]`

#### Scenario: Host without port is normalized

- **WHEN** the forwarder invokes `HostAudience.Resolve` on a request whose `URL.Scheme` is `https` and `URL.Host` is `Api.Example.COM`
- **THEN** the resolver returns `["https://api.example.com"]`

#### Scenario: IPv6 host with port is normalized

- **WHEN** the forwarder invokes `HostAudience.Resolve` on a request whose `URL.Scheme` is `https` and `URL.Host` is `[2001:db8::1]:8443`
- **THEN** the resolver returns `["https://[2001:db8::1]"]` with the port stripped and brackets preserved

#### Scenario: Scheme is preserved and lowercased

- **WHEN** the forwarder invokes `HostAudience.Resolve` on a request whose `URL.Scheme` is `http` or `HTTPS` and `URL.Host` is `api.example.com`
- **THEN** the resolver returns `["http://api.example.com"]` or `["https://api.example.com"]` respectively

#### Scenario: Empty host returns an error

- **WHEN** the forwarder invokes `HostAudience.Resolve` on a request whose `URL.Host` is empty
- **THEN** the resolver returns a non-nil error and the forwarder emits `502 Bad Gateway` via the existing `resolver_error` path with no STS call

#### Scenario: Empty scheme returns an error

- **WHEN** the forwarder invokes `HostAudience.Resolve` on a request whose `URL.Scheme` is empty
- **THEN** the resolver returns a non-nil error and the forwarder emits `502 Bad Gateway` via the existing `resolver_error` path with no STS call

### Requirement: Resolver selection based on configured audiences

The proxy SHALL select the `AudienceResolver` at startup based on `cfg.TokenAudiences`. If at least one audience is configured, the proxy SHALL use `token.StaticAudiences(cfg.TokenAudiences)`. Otherwise, the proxy SHALL use `token.HostAudience`. No CLI flag or environment variable SHALL control this selection directly.

#### Scenario: Static resolver used when audiences are configured

- **WHEN** the proxy starts with `--token-audience=https://api.example.com`
- **THEN** the constructed resolver is `StaticAudiences` and `Resolve` returns `["https://api.example.com"]` for every request regardless of target host

#### Scenario: Host resolver used when audiences are omitted

- **WHEN** the proxy starts with neither `--token-audience` nor `TOKEN_AUDIENCE` set
- **THEN** the constructed resolver is `HostAudience` and `Resolve` returns the normalized target host for each request

### Requirement: Cache and metric cardinality under host-derived audiences

The token cache and token metrics SHALL remain unchanged in shape; cache entries and `token.cached.audiences` observations SHALL naturally be keyed per distinct normalized host when `HostAudience` is active. No separate cache, counter, or gauge SHALL be introduced for the dynamic mode.

#### Scenario: Distinct hosts produce distinct cache entries

- **WHEN** the proxy forwards HTTPS requests to `api-a.example.com` and `api-b.example.com` in that order
- **THEN** the token cache contains two entries keyed on `https://api-a.example.com` and `https://api-b.example.com`, and the `token.cached.audiences` gauge reports `2`

#### Scenario: Equivalent hosts share one cache entry

- **WHEN** the proxy forwards an HTTPS request to `API.example.com:443` followed by an HTTPS request to `api.example.com`
- **THEN** the second request is served from the same cache entry as the first and `token.cached.audiences` reports `1`

#### Scenario: Token metric audience attribute carries the normalized audience

- **WHEN** a token is minted for an HTTPS request to `api.example.com:443`
- **THEN** the emitted `token.cache.miss.count` sample carries `audience=https://api.example.com`
