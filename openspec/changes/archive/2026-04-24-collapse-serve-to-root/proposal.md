## Why

The proxy is a single-purpose daemon. There is no anticipated second subcommand (operational tooling can live in flags or separate binaries), and requiring users to type `aws-outbound-jwt-proxy serve` adds noise to container images, systemd units, and Kubernetes specs.

## What Changes

- Collapse the `serve` subcommand into the root command: running `aws-outbound-jwt-proxy` (with no args) starts the server.
- Remove the `serve` command entry point.
- **BREAKING**: Anyone invoking `aws-outbound-jwt-proxy serve ...` must drop `serve`. Since there are no external users yet, this is cosmetic.

## Capabilities

### New Capabilities
<!-- None. This is a CLI-shape refactor. -->

### Modified Capabilities
- `proxy-configuration`: The entry-point contract changes from `<binary> serve [flags]` to `<binary> [flags]`.

## Impact

- Code: `cmd/root.go`, `cmd/serve.go` (removed/merged).
- No new dependencies.
- README or any docs referencing `serve` (none yet) would need updating.
- Behavior, flag names, env vars, and defaults are unchanged.
