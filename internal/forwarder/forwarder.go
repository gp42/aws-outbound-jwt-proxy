package forwarder

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/metrics"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/token"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type ctxKey int

const (
	targetKey ctxKey = iota
	startKey
	recordedKey
	tokenErrKey
)

// tokenFailure records which step of auth injection failed, so ErrorHandler
// can emit the right response and metric attribute.
type tokenFailure struct {
	kind string // "resolver_error" | "fetch_error"
	err  error
}

// WithTarget attaches a pre-resolved target URL to the request context.
func WithTarget(r *http.Request, target *url.URL) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), targetKey, target))
}

func Target(r *http.Request) *url.URL {
	v, _ := r.Context().Value(targetKey).(*url.URL)
	return v
}

// New builds an http.Handler that proxies requests to the target stored in
// the request context (via WithTarget), attaches an AWS-issued JWT via the
// given token.Source + AudienceResolver, and records OTel HTTP client
// metrics. tokenSource or resolver may be nil only in tests that do not
// exercise auth injection; production callers must supply both.
func New(cfg *config.Config, instruments *metrics.Instruments, tokenSource token.Source, resolver token.AudienceResolver) http.Handler {
	if instruments == nil {
		instruments = metrics.NoopInstruments()
	}

	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		DialContext:           (&net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: cfg.UpstreamTimeout,
	}

	stripHeader := ""
	if cfg.UpstreamHost == "" && !strings.EqualFold(cfg.HostHeader, "Host") {
		stripHeader = cfg.HostHeader
	}

	rp := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			target := Target(req)
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
			if stripHeader != "" {
				req.Header.Del(stripHeader)
			}
			if _, ok := req.Header["User-Agent"]; !ok {
				req.Header.Set("User-Agent", "")
			}

			if tokenSource == nil || resolver == nil {
				// Auth injection disabled (legacy test path). Leave the
				// request untouched.
				return
			}

			audiences, err := resolver.Resolve(req)
			if err != nil {
				stashTokenFailure(req, &tokenFailure{kind: "resolver_error", err: err})
				return
			}
			tok, err := tokenSource.Token(req.Context(), audiences)
			if err != nil {
				stashTokenFailure(req, &tokenFailure{kind: "fetch_error", err: err})
				return
			}
			if req.Header.Get("Authorization") != "" {
				slog.Debug("replacing inbound Authorization header",
					"target", target.String(),
				)
			}
			req.Header.Set("Authorization", "Bearer "+tok)
		},
		Transport:     transport,
		FlushInterval: -1,
		ModifyResponse: func(resp *http.Response) error {
			recordClient(resp.Request.Context(), instruments, Target(resp.Request), nil, "ok")
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, req *http.Request, err error) {
			if tf := takeTokenFailure(req); tf != nil {
				slog.Error("token injection failed",
					"kind", tf.kind,
					"target", urlString(Target(req)),
					"err", tf.err.Error(),
				)
				recordClient(req.Context(), instruments, Target(req), nil, tf.kind)
				w.WriteHeader(http.StatusBadGateway)
				_, _ = w.Write([]byte("token unavailable\n"))
				return
			}
			status := http.StatusBadGateway
			if isTimeout(err) {
				status = http.StatusGatewayTimeout
			}
			slog.Error("upstream error",
				"target", urlString(Target(req)),
				"status", status,
				"err", err.Error(),
			)
			recordClient(req.Context(), instruments, Target(req), err, "ok")
			w.WriteHeader(status)
		},
	}

	// Stamp start time on the request so recordClient can compute duration.
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := context.WithValue(req.Context(), startKey, time.Now())
		r := &recorded{}
		ctx = context.WithValue(ctx, recordedKey, r)
		ctx = context.WithValue(ctx, tokenErrKey, &tokenFailureSlot{})
		rp.ServeHTTP(w, req.WithContext(ctx))
	})
}

// tokenFailureSlot holds a token failure; stored in context by the entry
// handler and read/cleared by Director + ErrorHandler.
type tokenFailureSlot struct{ f *tokenFailure }

func stashTokenFailure(req *http.Request, tf *tokenFailure) {
	if slot, ok := req.Context().Value(tokenErrKey).(*tokenFailureSlot); ok {
		slot.f = tf
	}
	// Clear the URL host so ReverseProxy falls through to ErrorHandler
	// instead of attempting a dial. Director can't return an error, so this
	// is the idiomatic way to force the failure path.
	if req.URL != nil {
		req.URL.Host = ""
	}
}

func takeTokenFailure(req *http.Request) *tokenFailure {
	slot, ok := req.Context().Value(tokenErrKey).(*tokenFailureSlot)
	if !ok {
		return nil
	}
	tf := slot.f
	slot.f = nil
	return tf
}

// TokenFailure returns the token failure stashed on the request by the
// forwarder Director, or nil. Exposed for tests.
func TokenFailure(req *http.Request) error {
	tf := peekTokenFailure(req)
	if tf == nil {
		return nil
	}
	return tf.err
}

func peekTokenFailure(req *http.Request) *tokenFailure {
	slot, ok := req.Context().Value(tokenErrKey).(*tokenFailureSlot)
	if !ok {
		return nil
	}
	return slot.f
}

// recorded guards against double-recording if both ModifyResponse and
// ErrorHandler fire for the same request (they don't in practice, but the
// contract is not documented; belt and braces).
type recorded struct{ done bool }

func recordClient(ctx context.Context, instruments *metrics.Instruments, target *url.URL, err error, tokenResult string) {
	if r, ok := ctx.Value(recordedKey).(*recorded); ok {
		if r.done {
			return
		}
		r.done = true
	}
	start, _ := ctx.Value(startKey).(time.Time)
	if start.IsZero() {
		start = time.Now()
	}

	attrs := make([]attribute.KeyValue, 0, 4)
	host, port, emitPort := metrics.HostAndPort(target)
	if host != "" {
		attrs = append(attrs, semconv.ServerAddressKey.String(host))
	}
	if emitPort {
		attrs = append(attrs, semconv.ServerPortKey.Int(port))
	}
	if et, ok := metrics.ClassifyError(err); ok {
		attrs = append(attrs, semconv.ErrorTypeKey.String(et))
	}
	if tokenResult != "" {
		attrs = append(attrs, attribute.String("token.result", tokenResult))
	}

	instruments.HTTPClientRequestDuration.Record(ctx,
		time.Since(start).Seconds(),
		metric.WithAttributes(attrs...),
	)
}

func isTimeout(err error) bool {
	et, ok := metrics.ClassifyError(err)
	if !ok {
		return false
	}
	return et == "timeout"
}

func urlString(u *url.URL) string {
	if u == nil {
		return ""
	}
	return u.String()
}

