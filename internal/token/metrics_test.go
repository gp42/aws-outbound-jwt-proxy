package token

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func collect(t *testing.T, reader *sdkmetric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("collect: %v", err)
	}
	return rm
}

func sumCounter(rm metricdata.ResourceMetrics, name string, match map[string]string) int64 {
	var total int64
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			sum, ok := m.Data.(metricdata.Sum[int64])
			if !ok {
				continue
			}
		dp:
			for _, dp := range sum.DataPoints {
				for k, v := range match {
					got, ok := dp.Attributes.Value(attribute.Key(k))
					if !ok || got.AsString() != v {
						continue dp
					}
				}
				total += dp.Value
			}
		}
	}
	return total
}

func gaugeValue(rm metricdata.ResourceMetrics, name string) int64 {
	for _, sm := range rm.ScopeMetrics {
		for _, m := range sm.Metrics {
			if m.Name != name {
				continue
			}
			if g, ok := m.Data.(metricdata.Gauge[int64]); ok && len(g.DataPoints) > 0 {
				return g.DataPoints[0].Value
			}
		}
	}
	return -1
}

func TestMetrics(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	inst, err := NewInstruments(meter)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	stub := &stubSTS{handler: func(in *sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		return &sts.GetWebIdentityTokenOutput{
			WebIdentityToken: aws.String("t-" + in.Audience[0]),
			Expiration:       aws.Time(time.Now().Add(time.Hour)),
		}, nil
	}}
	cfg := &config.Config{TokenSigningAlgorithm: "RS256", TokenDuration: 15 * time.Minute, TokenRefreshSkew: 60 * time.Second}
	s := newWithClient(cfg, stub, inst, time.Now)

	// two audiences, four calls -> 2 misses + 2 fetches(ok) + 2 hits
	s.Token(context.Background(), []string{"A"})
	s.Token(context.Background(), []string{"A"})
	s.Token(context.Background(), []string{"B"})
	s.Token(context.Background(), []string{"B"})
	// plus a normalized multi-audience set, counted once in the gauge
	s.Token(context.Background(), []string{"B", "A"})
	s.Token(context.Background(), []string{"A", "B"})

	rm := collect(t, reader)
	// 2 hits for {A}, 2 hits for {B}, 1 hit for {A,B} (second call)
	if got := sumCounter(rm, "token.cache.hit.count", nil); got != 3 {
		t.Fatalf("cache hits: %d", got)
	}
	// 1 miss each for {A}, {B}, {A,B}
	if got := sumCounter(rm, "token.cache.miss.count", nil); got != 3 {
		t.Fatalf("cache misses: %d", got)
	}
	if got := sumCounter(rm, "token.fetch.count", map[string]string{"result": "ok"}); got != 3 {
		t.Fatalf("fetch ok: %d", got)
	}
	// 3 distinct normalized sets: {A}, {B}, {A,B}
	if got := gaugeValue(rm, "token.cached.audiences"); got != 3 {
		t.Fatalf("gauge: %d", got)
	}

	// per-audience hit label (normalized joined form)
	if got := sumCounter(rm, "token.cache.hit.count", map[string]string{"audience": "A"}); got != 1 {
		t.Fatalf("A hits: %d", got)
	}
	if got := sumCounter(rm, "token.cache.hit.count", map[string]string{"audience": "A,B"}); got != 1 {
		t.Fatalf("A,B hits: %d", got)
	}
}

func TestMetricsErrorClass(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	meter := mp.Meter("test")
	inst, err := NewInstruments(meter)
	if err != nil {
		t.Fatal(err)
	}
	defer inst.Close()

	stub := &stubSTS{handler: func(*sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		return nil, &apiErr{code: "AccessDenied", msg: "nope"}
	}}
	cfg := &config.Config{TokenSigningAlgorithm: "RS256", TokenDuration: 15 * time.Minute, TokenRefreshSkew: 60 * time.Second}
	s := newWithClient(cfg, stub, inst, time.Now)
	s.Token(context.Background(), []string{"A"})

	rm := collect(t, reader)
	if got := sumCounter(rm, "token.fetch.count", map[string]string{"result": "error", "error_class": "AccessDenied"}); got != 1 {
		t.Fatalf("AccessDenied fetch: %d", got)
	}
}
