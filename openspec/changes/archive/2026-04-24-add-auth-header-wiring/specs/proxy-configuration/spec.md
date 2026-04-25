## ADDED Requirements

### Requirement: Token audience flag (repeatable, required)

The proxy SHALL accept `--token-audience` (env `TOKEN_AUDIENCE`), supplying the audience values sent to AWS STS `GetWebIdentityToken` as the `Audience` array. The flag is repeatable on the command line; the environment variable is a comma-separated list. At least one non-empty value is REQUIRED. The proxy SHALL reject startup if zero values are supplied, if any value is empty, or if any value contains ASCII whitespace. There is no default.

#### Scenario: Single audience via flag

- **WHEN** the proxy is started with `--token-audience=https://api.example.com`
- **THEN** the configured audience set is `["https://api.example.com"]` and this slice is what the forwarder's default `AudienceResolver` returns for every request

#### Scenario: Multiple audiences via repeated flag

- **WHEN** the proxy is started with `--token-audience=https://a.example.com --token-audience=https://b.example.com`
- **THEN** the configured audience set is `["https://a.example.com", "https://b.example.com"]`

#### Scenario: Multiple audiences via env

- **WHEN** `TOKEN_AUDIENCE=https://a.example.com,https://b.example.com` is set and no CLI flag is passed
- **THEN** the configured audience set is `["https://a.example.com", "https://b.example.com"]`

#### Scenario: CLI flag precedence over env

- **WHEN** `TOKEN_AUDIENCE=https://env.example.com` is set AND `--token-audience=https://cli.example.com` is passed
- **THEN** the configured audience set is `["https://cli.example.com"]` (the env value is ignored, matching existing flag-vs-env precedence rules)

#### Scenario: Missing audience rejected

- **WHEN** the proxy is started with no `--token-audience` flag and no `TOKEN_AUDIENCE` env var
- **THEN** the proxy exits with a non-zero status and an error naming `--token-audience` as required

#### Scenario: Empty string audience rejected

- **WHEN** the proxy is started with `TOKEN_AUDIENCE=https://a.example.com,,https://b.example.com`
- **THEN** the proxy exits with a non-zero status and an error stating audience values must be non-empty

#### Scenario: Whitespace in audience rejected

- **WHEN** the proxy is started with `--token-audience="api example.com"`
- **THEN** the proxy exits with a non-zero status and an error stating audience values must not contain whitespace
