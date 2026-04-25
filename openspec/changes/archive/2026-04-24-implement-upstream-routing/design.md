## Context

The repository has no application code yet beyond a placeholder `main.go`. This change lays down the package layout and implements only the upstream-routing capability (plus the slice of proxy-configuration it depends on). The rest of the architecture (token acquisition, forwarding, TLS) is deliberately stubbed so we can validate the config surface and routing precedence in isolation.

Routing rules are already agreed in `proposal.md` and the parent `initial-proxy-architecture` change: pinned upstream wins; otherwise read from a configurable header (default `X-Upstream-Host`).

## Goals / Non-Goals

**Goals:**
- Establish package layout: `cmd/proxy/`, `internal/config/`, `internal/router/`, `internal/server/`.
- Implement a `Resolver` that returns the target `*url.URL` for an incoming request, honoring the pinned-wins precedence.
- Load config from CLI flags with env-var fallback (CLI wins). Each flag has exactly one corresponding env var with a mechanical mapping (`--upstream-host` ↔ `UPSTREAM_HOST`).
- Accept any inbound HTTP request, resolve target, debug-print it, respond `204`. Return `400` if no upstream can be resolved.

**Non-Goals:**
- Token acquisition, JWT injection, request forwarding — later changes.
- TLS server mode — later change.
- Path/query rewriting, header filtering, response streaming — N/A while stubbed.
- Metrics, structured logging, graceful shutdown polish — out of scope; a minimal `log.Printf` is enough.

## Decisions

**Flag parsing: `spf13/cobra` + manual env-fallback helper. No viper yet.**
Cobra is the de-facto standard for production Go server daemons (Kubernetes, Helm, Caddy, etcd, Docker, GitHub CLI). The proxy is a long-lived server that will grow subcommands (`version`, later possibly `healthcheck`, `inspect-token`), so the framework pays off.

Viper is intentionally deferred: it adds ~7 MB to the binary (parsers for YAML/TOML/HCL/INI/JSON/properties plus fsnotify). We only need env-var fallback right now, which is a ~15-line helper. If config-file support becomes useful later, swapping in viper is a localized change because cobra is already in place.

Helper shape: after cobra parses flags, walk the command's flagset with `cmd.Flags().VisitAll`; for any flag whose `Changed` is false, look up the corresponding env var (`strings.ReplaceAll(strings.ToUpper(name), "-", "_")`) and call `flag.Value.Set(envVal)` if present. CLI-wins is inherent: we only consult env for unset flags.

The existing scaffold (`main.go` → `cmd/root.go` + `cmd/serve.go`) is kept. The `serve` subcommand builds a `Config` via the helper and hands it to `internal/server`.

**Resolver signature: `Resolve(r *http.Request) (*url.URL, error)`.**
Returns the full target URL (scheme + host + original request path + raw query). The handler only needs one call to get a forwardable target. When neither pinned nor header is set, returns a sentinel error that maps to `400`.

Alternative: return `(scheme, host string, err error)` and have the caller assemble. Rejected — pushing URL assembly to callers invites bugs when the forwarder is added.

**Header read uses canonical form; `Host` is special-cased.**
If `--host-header` resolves to `Host` (case-insensitive), read from `r.Host` rather than `r.Header` because Go's server moves the Host header there. For any other header name, use `r.Header.Get(name)` which handles canonicalization.

**Upstream-scheme default: `https`.**
Almost all SaaS targets are HTTPS. `http` is available as an explicit opt-in.

**Pinned-mode host-header behavior: ignore the header entirely when pinned is set.**
No "pin unless header overrides" mode. The proposal explicitly chose pinned-wins to prevent client-side redirection. The header is not read in pinned mode.

**Stub response: `204 No Content`, debug line to stdout via `log.Printf`.**
`204` signals "I received it, there's no body" without implying success of a downstream call. The debug line format is `resolved upstream: <scheme>://<host><path>` — just enough to verify routing end-to-end by curl.

## Risks / Trade-offs

- [Risk] Users set `--upstream-host` to a full URL like `https://api.example.com` by mistake. → Validate at config load: reject values containing `://` with a clear error pointing them at `--upstream-scheme`.
- [Risk] Header-mode clients forget to set the header and get confusing 400s. → Error body says which header was expected (the configured name) so the mistake is self-evident in logs/curl output.
- [Risk] `--host-header=Host` footgun — some HTTP clients silently overwrite Host from the URL. → Documented in proposal; not a code concern, but the `Host`-special-case in the resolver means the proxy behaves correctly if the client *does* manage to set it.
- [Trade-off] Hand-rolled env fallback means no config-file support and no typed env parsing. Acceptable while the surface is small; the helper is self-contained and easy to replace.

## Open Questions

- Should `--upstream-scheme` accept only `http`/`https`, or allow arbitrary schemes for future-proofing? Leaning toward whitelist (reject unknown schemes early); confirm during implementation.
