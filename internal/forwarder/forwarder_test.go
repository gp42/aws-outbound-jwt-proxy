package forwarder

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/metrics"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/token"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/token/tokentest"
)

func newProvider(t *testing.T) (*metrics.Provider, *metrics.Instruments) {
	t.Helper()
	p, err := metrics.New(&config.Config{MetricsEnabled: true, MetricsListenAddr: ":0", MetricsPath: "/metrics"})
	if err != nil {
		t.Fatalf("metrics.New: %v", err)
	}
	inst, err := p.Instruments()
	if err != nil {
		t.Fatalf("Instruments: %v", err)
	}
	return p, inst
}

func scrape(t *testing.T, p *metrics.Provider) string {
	t.Helper()
	srv := httptest.NewServer(p.Handler("/metrics"))
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return string(body)
}

func TestForwarderSuccessNoErrorType(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	h := New(&config.Config{UpstreamTimeout: time.Second}, inst, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, WithTarget(httptest.NewRequest("GET", "/x", nil), u))

	body := scrape(t, p)
	if !strings.Contains(body, `http_client_request_duration_seconds_count{`) {
		t.Fatalf("missing client duration count:\n%s", body)
	}
	if strings.Contains(body, `error_type=`) {
		t.Fatalf("success path must not emit error.type:\n%s", body)
	}
}

func TestForwarderConnectionRefused(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close() // close -> subsequent connect is refused

	u, _ := url.Parse("http://" + addr)

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	h := New(&config.Config{UpstreamTimeout: time.Second}, inst, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, WithTarget(httptest.NewRequest("GET", "/x", nil), u))
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d", rec.Code)
	}

	body := scrape(t, p)
	if !strings.Contains(body, `error_type="connection_refused"`) {
		t.Fatalf("expected connection_refused:\n%s", body)
	}
}

func TestForwarderTimeout(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	h := New(&config.Config{UpstreamTimeout: 20 * time.Millisecond}, inst, nil, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, WithTarget(httptest.NewRequest("GET", "/x", nil), u))
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status %d", rec.Code)
	}

	body := scrape(t, p)
	if !strings.Contains(body, `error_type="timeout"`) {
		t.Fatalf("expected timeout:\n%s", body)
	}
}

func TestForwarderAttachesBearerToken(t *testing.T) {
	var seenAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	src := tokentest.New(map[string]string{"https://api.example.com": "jwt-abc"})
	h := New(&config.Config{UpstreamTimeout: time.Second}, inst, src, token.StaticAudiences([]string{"https://api.example.com"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, WithTarget(httptest.NewRequest("GET", "/x", nil), u))

	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	if seenAuth != "Bearer jwt-abc" {
		t.Fatalf("upstream saw Authorization=%q", seenAuth)
	}

	body := scrape(t, p)
	if !strings.Contains(body, `token_result="ok"`) {
		t.Fatalf("expected token_result=ok:\n%s", body)
	}
}

func TestForwarderReplacesInboundAuthorization(t *testing.T) {
	var seenAuth string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	_, inst := newProvider(t)

	src := tokentest.New(map[string]string{"aud-1": "jwt-new"})
	h := New(&config.Config{UpstreamTimeout: time.Second}, inst, src, token.StaticAudiences([]string{"aud-1"}))
	req := httptest.NewRequest("GET", "/x", nil)
	req.Header.Set("Authorization", "Basic dXNlcjpwdw==")
	h.ServeHTTP(httptest.NewRecorder(), WithTarget(req, u))

	if seenAuth != "Bearer jwt-new" {
		t.Fatalf("expected replaced bearer, got %q", seenAuth)
	}
}

func TestForwarderTokenFetchErrorReturns502(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalled = true
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	src := tokentest.New(map[string]string{"known": "t"})
	h := New(&config.Config{UpstreamTimeout: time.Second}, inst, src, token.StaticAudiences([]string{"unknown-audience-should-fail"}))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, WithTarget(httptest.NewRequest("GET", "/x", nil), u))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d", rec.Code)
	}
	if upstreamCalled {
		t.Fatalf("upstream must not be called on token fetch failure")
	}
	body, _ := io.ReadAll(rec.Body)
	if string(body) != "token unavailable\n" {
		t.Fatalf("body=%q", body)
	}
	if strings.Contains(string(body), "no token configured") {
		t.Fatalf("body leaked token source detail: %q", body)
	}

	m := scrape(t, p)
	if !strings.Contains(m, `token_result="fetch_error"`) {
		t.Fatalf("expected token_result=fetch_error:\n%s", m)
	}
}

type errResolver struct{ err error }

func (e errResolver) Resolve(*http.Request) ([]string, error) { return nil, e.err }

func TestForwarderResolverErrorReturns502(t *testing.T) {
	upstreamCalled := false
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		upstreamCalled = true
		w.WriteHeader(200)
	}))
	defer upstream.Close()
	u, _ := url.Parse(upstream.URL)

	p, inst := newProvider(t)
	defer p.Shutdown(context.Background())

	src := tokentest.New(map[string]string{"anything": "t"})
	h := New(&config.Config{UpstreamTimeout: time.Second}, inst, src, errResolver{err: errors.New("no audience")})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, WithTarget(httptest.NewRequest("GET", "/x", nil), u))

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status %d", rec.Code)
	}
	if upstreamCalled {
		t.Fatalf("upstream must not be called on resolver failure")
	}
	m := scrape(t, p)
	if !strings.Contains(m, `token_result="resolver_error"`) {
		t.Fatalf("expected token_result=resolver_error:\n%s", m)
	}
}
