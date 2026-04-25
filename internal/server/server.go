package server

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/forwarder"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/metrics"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/router"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/token"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func New(cfg *config.Config, instruments *metrics.Instruments, tokenSource token.Source, audienceResolver token.AudienceResolver) *http.Server {
	if instruments == nil {
		instruments = metrics.NoopInstruments()
	}
	resolver := router.New(cfg.UpstreamHost, cfg.UpstreamScheme, cfg.HostHeader)
	fwd := forwarder.New(cfg, instruments, tokenSource, audienceResolver)
	return &http.Server{
		Addr:    cfg.ListenAddr,
		Handler: handler(resolver, fwd, instruments),
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func handler(r *router.Resolver, fwd http.Handler, instruments *metrics.Instruments) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		start := time.Now()
		ctx := req.Context()

		instruments.HTTPServerActiveRequests.Add(ctx, 1)
		defer instruments.HTTPServerActiveRequests.Add(ctx, -1)

		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		target, err := r.Resolve(req)
		if err != nil {
			if errors.Is(err, router.ErrNoUpstream) {
				rec.WriteHeader(http.StatusBadRequest)
				fmt.Fprintf(rec, "missing upstream: set --upstream-host or provide %s\n", r.HeaderName())
			} else {
				http.Error(rec, err.Error(), http.StatusInternalServerError)
			}
			recordServerRequest(ctx, instruments, req.Method, rec.status, start)
			return
		}

		fwd.ServeHTTP(rec, forwarder.WithTarget(req, target))
		recordServerRequest(ctx, instruments, req.Method, rec.status, start)

		slog.Debug("request",
			"method", req.Method,
			"path", req.URL.RequestURI(),
			"target", target.String(),
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func recordServerRequest(ctx context.Context, instruments *metrics.Instruments, method string, status int, start time.Time) {
	attrs := metric.WithAttributes(
		semconv.HTTPRequestMethodKey.String(metrics.CanonicalMethod(method)),
		semconv.HTTPResponseStatusCodeKey.Int(status),
	)
	instruments.HTTPServerRequestDuration.Record(ctx, time.Since(start).Seconds(), attrs)
}
