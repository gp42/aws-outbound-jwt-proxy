package metrics

import (
	"context"
	"errors"
	"net"
	"net/url"
	"syscall"
	"testing"
)

func TestCanonicalMethod(t *testing.T) {
	cases := map[string]string{
		"GET":     "GET",
		"get":     "GET",
		"POST":    "POST",
		"OPTIONS": "OPTIONS",
		"PATCH":   "PATCH",
		"FOO":     "_OTHER",
		"":        "_OTHER",
		"propfind": "_OTHER",
	}
	for in, want := range cases {
		if got := CanonicalMethod(in); got != want {
			t.Errorf("CanonicalMethod(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestHostAndPort(t *testing.T) {
	cases := []struct {
		raw       string
		host      string
		port      int
		emitPort  bool
	}{
		{"https://api.example.com/x", "api.example.com", 443, false},
		{"http://api.example.com/x", "api.example.com", 80, false},
		{"https://api.example.com:8443/x", "api.example.com", 8443, true},
		{"http://api.example.com:8080/x", "api.example.com", 8080, true},
	}
	for _, tc := range cases {
		u, _ := url.Parse(tc.raw)
		host, port, emit := HostAndPort(u)
		if host != tc.host || port != tc.port || emit != tc.emitPort {
			t.Errorf("HostAndPort(%q)=(%q,%d,%v), want (%q,%d,%v)",
				tc.raw, host, port, emit, tc.host, tc.port, tc.emitPort)
		}
	}
	if h, _, emit := HostAndPort(nil); h != "" || emit {
		t.Errorf("nil URL: got (%q, _, %v)", h, emit)
	}
}

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return false }

func TestClassifyError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want string
		ok   bool
	}{
		{"nil", nil, "", false},
		{"deadline", context.DeadlineExceeded, "timeout", true},
		{"net.Error timeout", timeoutErr{}, "timeout", true},
		{"connection refused", syscall.ECONNREFUSED, "connection_refused", true},
		{"wrapped refused", &net.OpError{Op: "dial", Err: syscall.ECONNREFUSED}, "connection_refused", true},
		{"other", errors.New("boom"), "unknown", true},
	}
	for _, tc := range cases {
		got, ok := ClassifyError(tc.err)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: got (%q, %v), want (%q, %v)", tc.name, got, ok, tc.want, tc.ok)
		}
	}
}
