## REMOVED Requirements

### Requirement: Stub forwarding behavior
**Reason**: Replaced by real streaming forwarding in the `request-forwarding` capability introduced by this change.
**Migration**: N/A — internal behavior; no external consumers. The `204 No Content` stub is replaced by the upstream's actual response (or `400` / `502` / `504`).
