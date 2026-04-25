package version

// Version is the build-time version string. It is overridden via
// -ldflags "-X github.com/gp42/aws-outbound-jwt-proxy/internal/version.Version=..."
// in release builds. Local builds keep the default.
var Version = "dev"

func String() string {
	return Version
}
