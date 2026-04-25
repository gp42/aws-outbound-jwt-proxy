package metrics

import "runtime/debug"

// moduleVersion reports the version string to attach as the instrumentation
// scope version. Preference order:
//  1. Main module version (e.g. "v0.3.1") when built from a tagged module.
//  2. Short VCS revision when available (Go records this with -buildvcs=true,
//     which is the default).
//  3. "(devel)" as a last-resort sentinel.
func moduleVersion() string {
	return resolveVersion(debug.ReadBuildInfo)
}

func resolveVersion(read func() (*debug.BuildInfo, bool)) string {
	info, ok := read()
	if !ok {
		return "(devel)"
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	for _, s := range info.Settings {
		if s.Key == "vcs.revision" && s.Value != "" {
			if len(s.Value) > 12 {
				return s.Value[:12]
			}
			return s.Value
		}
	}
	return "(devel)"
}
