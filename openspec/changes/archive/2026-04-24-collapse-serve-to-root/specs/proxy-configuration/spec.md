## ADDED Requirements

### Requirement: Server starts from root command

The proxy SHALL start its HTTP server when invoked as the root command with no subcommand. Flags and environment variables defined by proxy-configuration SHALL apply to the root command.

#### Scenario: Invoking without a subcommand starts the server

- **WHEN** the user runs `aws-outbound-jwt-proxy --upstream-host=api.example.com`
- **THEN** the server starts listening on the configured address with the given upstream host

#### Scenario: No dedicated serve subcommand

- **WHEN** the user runs `aws-outbound-jwt-proxy serve`
- **THEN** cobra reports an unknown command error
