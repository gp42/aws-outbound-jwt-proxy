## MODIFIED Requirements

### Requirement: Token audience flag (repeatable, optional)

The proxy SHALL accept `--token-audience` (env `TOKEN_AUDIENCE`), supplying the audience values sent to AWS STS `GetWebIdentityToken` as the `Audience` array. The flag is repeatable on the command line; the environment variable is a comma-separated list. The flag is OPTIONAL: if no value is supplied, the proxy SHALL start and select the `HostAudience` resolver so the audience is derived per-request from the outbound target host. When values ARE supplied, the proxy SHALL still reject any empty value and any value containing ASCII whitespace. There is no default audience string.

#### Scenario: Single audience via flag

- **WHEN** the proxy is started with `--token-audience=https://api.example.com`
- **THEN** the configured audience set is `["https://api.example.com"]`, the resolver is `StaticAudiences`, and every request uses that slice

#### Scenario: Multiple audiences via repeated flag

- **WHEN** the proxy is started with `--token-audience=https://a.example.com --token-audience=https://b.example.com`
- **THEN** the configured audience set is `["https://a.example.com", "https://b.example.com"]` and the resolver is `StaticAudiences`

#### Scenario: Multiple audiences via env

- **WHEN** `TOKEN_AUDIENCE=https://a.example.com,https://b.example.com` is set and no CLI flag is passed
- **THEN** the configured audience set is `["https://a.example.com", "https://b.example.com"]` and the resolver is `StaticAudiences`

#### Scenario: CLI flag precedence over env

- **WHEN** `TOKEN_AUDIENCE=https://env.example.com` is set AND `--token-audience=https://cli.example.com` is passed
- **THEN** the configured audience set is `["https://cli.example.com"]` (the env value is ignored, matching existing flag-vs-env precedence rules)

#### Scenario: Missing audience selects the host-derived resolver

- **WHEN** the proxy is started with no `--token-audience` flag and no `TOKEN_AUDIENCE` env var
- **THEN** the proxy starts successfully, the configured audience set is empty, and the constructed resolver is `HostAudience`

#### Scenario: Empty string audience rejected

- **WHEN** the proxy is started with `TOKEN_AUDIENCE=https://a.example.com,,https://b.example.com`
- **THEN** the proxy exits with a non-zero status and an error stating audience values must be non-empty

#### Scenario: Whitespace in audience rejected

- **WHEN** the proxy is started with `--token-audience="api example.com"`
- **THEN** the proxy exits with a non-zero status and an error stating audience values must not contain whitespace
