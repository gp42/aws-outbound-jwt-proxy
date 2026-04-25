# Runtime image for aws-outbound-jwt-proxy.
#
# This Dockerfile does NOT compile Go. It expects the per-architecture
# binary to already be staged at dist/linux/<arch>/aws-outbound-jwt-proxy
# (produced by the release workflow's build matrix and downloaded by the
# release workflow's docker job). Building from prebuilt artifacts is
# deliberate: it guarantees the bytes shipped on Docker Hub / GHCR are
# byte-identical to the asset attached to the GitHub Release.
#
# Base image: distroless/static-debian13:nonroot — pinned by digest.
# Tag (for human readers; do not rely on it at build time): nonroot
FROM gcr.io/distroless/static-debian13@sha256:e3f945647ffb95b5839c07038d64f9811adf17308b9121d8a2b87b6a22a80a39

ARG TARGETARCH
COPY dist/linux/${TARGETARCH}/aws-outbound-jwt-proxy /usr/local/bin/aws-outbound-jwt-proxy

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/aws-outbound-jwt-proxy"]
