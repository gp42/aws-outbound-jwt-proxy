package metrics

import (
	"runtime/debug"
	"testing"
)

func TestResolveVersionUsesMainVersion(t *testing.T) {
	got := resolveVersion(func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v1.2.3"}}, true
	})
	if got != "v1.2.3" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveVersionFallsBackToVCSRevision(t *testing.T) {
	got := resolveVersion(func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main:     debug.Module{Version: "(devel)"},
			Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abcdef0123456789deadbeef"}},
		}, true
	})
	if got != "abcdef012345" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveVersionShortVCSRevision(t *testing.T) {
	got := resolveVersion(func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{
			Main:     debug.Module{Version: ""},
			Settings: []debug.BuildSetting{{Key: "vcs.revision", Value: "abc123"}},
		}, true
	})
	if got != "abc123" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveVersionDevelSentinel(t *testing.T) {
	got := resolveVersion(func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	})
	if got != "(devel)" {
		t.Fatalf("got %q", got)
	}
}

func TestResolveVersionNoBuildInfo(t *testing.T) {
	got := resolveVersion(func() (*debug.BuildInfo, bool) { return nil, false })
	if got != "(devel)" {
		t.Fatalf("got %q", got)
	}
}
