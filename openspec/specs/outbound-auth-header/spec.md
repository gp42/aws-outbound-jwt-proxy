# outbound-auth-header Specification

## Purpose
TBD - created by archiving change add-auth-header-wiring. Update Purpose after archive.
## Requirements
### Requirement: Attach AWS-issued JWT as bearer token on every forwarded request

The proxy SHALL set the outbound request's `Authorization` header to `Bearer <jwt>`, where `<jwt>` is obtained from `token.Source.Token` using the audience set returned by the configured `AudienceResolver` for that request. The header SHALL be set before the request leaves the process and SHALL replace any `Authorization` header present on the inbound request.

#### Scenario: Token attached on successful fetch

- **WHEN** a client sends a request through the proxy, the resolver returns audience set `A`, and `token.Source.Token(ctx, A)` returns a JWT `T`
- **THEN** the forwarded upstream request carries `Authorization: Bearer T`

#### Scenario: Inbound Authorization is overwritten

- **WHEN** the inbound request carries `Authorization: Basic dXNlcjpwdw==` and the token fetch returns a JWT `T`
- **THEN** the forwarded upstream request carries `Authorization: Bearer T` and no part of the inbound credential is transmitted

#### Scenario: Overwriting an inbound Authorization is logged at debug

- **WHEN** the inbound request carries a non-empty `Authorization` header that gets replaced
- **THEN** the proxy emits a `debug`-level log entry noting the overwrite (without logging the original header value)

### Requirement: Fail the request cleanly when the token cannot be obtained

The proxy SHALL NOT forward a request upstream if either `AudienceResolver.Resolve` or `token.Source.Token` returns an error. It SHALL respond to the client with HTTP `502 Bad Gateway` and a short, non-sensitive error body, and SHALL log the underlying error at `error` level.

#### Scenario: Token fetch error returns 502 and skips upstream

- **WHEN** `token.Source.Token` returns an error for the request's audience set
- **THEN** the proxy responds `502 Bad Gateway`, no request is sent to the upstream, and the underlying error (including wrapped AWS error type) is logged at `error`

#### Scenario: Audience resolver error returns 502

- **WHEN** `AudienceResolver.Resolve` returns an error for the inbound request
- **THEN** the proxy responds `502 Bad Gateway`, no token fetch is attempted, and the resolver error is logged at `error`

#### Scenario: Error body does not leak AWS detail

- **WHEN** the proxy responds `502` because of a token error
- **THEN** the response body contains a fixed short message (e.g. "token unavailable") and does NOT contain the AWS error message, error code details, request ID, or stack trace

### Requirement: Token-result visibility in HTTP client metrics

The proxy SHALL record an attribute `token.result` on the `http.client.*` instruments for every request that enters the forwarder, with values `ok`, `fetch_error`, or `resolver_error`.

#### Scenario: Successful fetch records token.result=ok

- **WHEN** the token is obtained successfully and the upstream is called
- **THEN** the `http.client.request.duration` sample carries `token.result=ok`

#### Scenario: Token fetch failure records token.result=fetch_error

- **WHEN** `token.Source.Token` returns an error
- **THEN** the `http.client.request.duration` sample carries `token.result=fetch_error` and `http.response.status_code=502`

#### Scenario: Resolver failure records token.result=resolver_error

- **WHEN** `AudienceResolver.Resolve` returns an error
- **THEN** the `http.client.request.duration` sample carries `token.result=resolver_error` and `http.response.status_code=502`

### Requirement: Audience resolution is per-request via the AudienceResolver seam

The forwarder SHALL obtain the audience set for each outbound request by calling `AudienceResolver.Resolve(req)` and SHALL NOT cache or capture the audience set at forwarder construction time. This keeps the injection point compatible with future dynamic (e.g. per-host) resolvers without changing the forwarder.

#### Scenario: Resolver is called for every request

- **WHEN** the proxy receives two consecutive requests
- **THEN** `AudienceResolver.Resolve` is invoked once per request, not once at startup

#### Scenario: Resolver return value is passed verbatim to the token source

- **WHEN** `AudienceResolver.Resolve` returns slice `S` for a request
- **THEN** the forwarder calls `token.Source.Token(ctx, S)` — the slice is passed by value without filtering, sorting, or deduplication inside the forwarder (normalization is the token source's responsibility)

