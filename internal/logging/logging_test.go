package logging

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
)

func TestJSONOutputAtInfoFiltersDebug(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&config.Config{LogLevel: "info", LogFormat: "json"}, &buf)
	l.Debug("not shown")
	l.Info("shown", "k", "v")

	lines := splitNonEmpty(buf.String())
	if len(lines) != 1 {
		t.Fatalf("want 1 line, got %d: %q", len(lines), buf.String())
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &rec); err != nil {
		t.Fatalf("not json: %v", err)
	}
	if rec["msg"] != "shown" || rec["k"] != "v" {
		t.Fatalf("bad record: %v", rec)
	}
}

func TestTextFormat(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&config.Config{LogLevel: "info", LogFormat: "text"}, &buf)
	l.Info("hello", "k", "v")
	out := buf.String()
	if !strings.Contains(out, "msg=hello") || !strings.Contains(out, "k=v") {
		t.Fatalf("unexpected text output: %q", out)
	}
}

func TestDebugLevelShowsDebug(t *testing.T) {
	var buf bytes.Buffer
	l := NewWithWriter(&config.Config{LogLevel: "debug", LogFormat: "json"}, &buf)
	l.Debug("shown")
	if !strings.Contains(buf.String(), "shown") {
		t.Fatalf("debug missing: %q", buf.String())
	}
}

func splitNonEmpty(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}
