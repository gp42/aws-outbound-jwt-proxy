## ADDED Requirements

### Requirement: Token signing algorithm flag

The proxy SHALL accept `--token-signing-algorithm` (env `TOKEN_SIGNING_ALGORITHM`), specifying the signing algorithm sent to AWS STS `GetWebIdentityToken`. Valid values are `RS256` and `ES384`. Default `RS256`. Any other value SHALL be rejected at startup.

#### Scenario: Default algorithm

- **WHEN** the proxy is started without `--token-signing-algorithm`
- **THEN** the effective signing algorithm is `RS256`

#### Scenario: ES384 via flag

- **WHEN** the proxy is started with `--token-signing-algorithm=ES384`
- **THEN** the effective signing algorithm is `ES384`

#### Scenario: Invalid value rejected

- **WHEN** the proxy is started with `--token-signing-algorithm=HS256`
- **THEN** the proxy exits with a non-zero status and an error naming the invalid value and listing the valid options

### Requirement: Token duration flag

The proxy SHALL accept `--token-duration` (env `TOKEN_DURATION`), a Go duration string bounding the requested token lifetime sent as `DurationSeconds`. The value SHALL be in the range `[60s, 3600s]` inclusive. Default `900s`. Values outside the range SHALL be rejected at startup.

#### Scenario: Default duration

- **WHEN** the proxy is started without `--token-duration`
- **THEN** the effective token duration is `900s`

#### Scenario: Custom duration via env

- **WHEN** `TOKEN_DURATION=300s` is set and no CLI flag is passed
- **THEN** the effective token duration is `300s`

#### Scenario: Below minimum rejected

- **WHEN** the proxy is started with `--token-duration=30s`
- **THEN** the proxy exits with a non-zero status and an error stating the value must be between `60s` and `3600s`

#### Scenario: Above maximum rejected

- **WHEN** the proxy is started with `--token-duration=2h`
- **THEN** the proxy exits with a non-zero status and an error stating the value must be between `60s` and `3600s`

### Requirement: Token refresh skew flag

The proxy SHALL accept `--token-refresh-skew` (env `TOKEN_REFRESH_SKEW`), a Go duration string defining how far in advance of expiry a cached token SHALL be considered stale. The value SHALL be strictly greater than `0` and strictly less than the effective `--token-duration`. Default `60s`. Violations SHALL be rejected at startup.

#### Scenario: Default skew

- **WHEN** the proxy is started without `--token-refresh-skew`
- **THEN** the effective refresh skew is `60s`

#### Scenario: Zero skew rejected

- **WHEN** the proxy is started with `--token-refresh-skew=0`
- **THEN** the proxy exits with a non-zero status and an error stating the value must be positive

#### Scenario: Skew not less than duration rejected

- **WHEN** the proxy is started with `--token-duration=60s` and `--token-refresh-skew=60s`
- **THEN** the proxy exits with a non-zero status and an error stating the skew must be strictly less than the token duration
