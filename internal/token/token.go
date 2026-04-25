// Package token acquires and caches AWS-issued JWTs via the STS
// GetWebIdentityToken API for use as outbound bearer tokens.
package token

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
	"go.opentelemetry.io/otel/metric"
	"golang.org/x/sync/singleflight"
)

// Source returns a currently-valid JWT whose `aud` claim carries the given
// audience set. Callers may pass audiences in any order; the implementation
// normalizes (sort + dedupe) before keying the cache and calling STS.
type Source interface {
	Token(ctx context.Context, audiences []string) (string, error)
}

// stsClient is the subset of the STS SDK surface the cache uses. Tests stub it.
type stsClient interface {
	GetWebIdentityToken(ctx context.Context, in *sts.GetWebIdentityTokenInput, optFns ...func(*sts.Options)) (*sts.GetWebIdentityTokenOutput, error)
}

// Instruments groups the token-related OTel instruments.
type Instruments struct {
	FetchCount  metric.Int64Counter
	CacheHit    metric.Int64Counter
	CacheMiss   metric.Int64Counter
	CachedGauge metric.Int64ObservableGauge
	gaugeReg    metric.Registration
	cachedSize  *atomic.Int64
}

// NewInstruments registers the token.* instruments on the given meter. The
// observable gauge reads cachedSize, which the source updates atomically.
func NewInstruments(meter metric.Meter) (*Instruments, error) {
	fetch, err := meter.Int64Counter("token.fetch.count", metric.WithDescription("Count of AWS STS GetWebIdentityToken calls."))
	if err != nil {
		return nil, fmt.Errorf("token.fetch.count: %w", err)
	}
	hit, err := meter.Int64Counter("token.cache.hit.count", metric.WithDescription("Count of token requests served from cache."))
	if err != nil {
		return nil, fmt.Errorf("token.cache.hit.count: %w", err)
	}
	miss, err := meter.Int64Counter("token.cache.miss.count", metric.WithDescription("Count of token requests that missed the cache."))
	if err != nil {
		return nil, fmt.Errorf("token.cache.miss.count: %w", err)
	}
	size := &atomic.Int64{}
	gauge, err := meter.Int64ObservableGauge("token.cached.audiences", metric.WithDescription("Number of distinct audience sets currently cached."))
	if err != nil {
		return nil, fmt.Errorf("token.cached.audiences: %w", err)
	}
	reg, err := meter.RegisterCallback(func(_ context.Context, o metric.Observer) error {
		o.ObserveInt64(gauge, size.Load())
		return nil
	}, gauge)
	if err != nil {
		return nil, fmt.Errorf("token.cached.audiences callback: %w", err)
	}
	return &Instruments{
		FetchCount:  fetch,
		CacheHit:    hit,
		CacheMiss:   miss,
		CachedGauge: gauge,
		gaugeReg:    reg,
		cachedSize:  size,
	}, nil
}

// Close unregisters the observable-gauge callback. Safe to call more than once.
func (i *Instruments) Close() error {
	if i == nil || i.gaugeReg == nil {
		return nil
	}
	reg := i.gaugeReg
	i.gaugeReg = nil
	return reg.Unregister()
}

type cacheEntry struct {
	token string
	exp   time.Time
}

type source struct {
	client   stsClient
	algo     string
	duration int32
	skew     time.Duration
	now      func() time.Time
	cache    sync.Map // joined-audience-key -> *cacheEntry
	sf       singleflight.Group
	inst     *Instruments
}

// New builds a Source backed by AWS STS using the default AWS SDK config chain
// for region and credentials.
func New(ctx context.Context, cfg *config.Config, inst *Instruments) (Source, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("token: load aws config: %w", err)
	}
	return newWithClient(cfg, sts.NewFromConfig(awsCfg), inst, time.Now), nil
}

func newWithClient(cfg *config.Config, client stsClient, inst *Instruments, now func() time.Time) *source {
	return &source{
		client:   client,
		algo:     cfg.TokenSigningAlgorithm,
		duration: int32(cfg.TokenDuration / time.Second),
		skew:     cfg.TokenRefreshSkew,
		now:      now,
		inst:     inst,
	}
}

// normalizeAudiences sorts and dedupes the input slice and returns the
// normalized slice plus its comma-joined form, used as both cache key and
// metric attribute value. Empty or nil input is an error.
func normalizeAudiences(audiences []string) ([]string, string, error) {
	if len(audiences) == 0 {
		return nil, "", errors.New("token: audiences must be non-empty")
	}
	sorted := make([]string, len(audiences))
	copy(sorted, audiences)
	sort.Strings(sorted)
	out := sorted[:0]
	for i, a := range sorted {
		if i > 0 && a == sorted[i-1] {
			continue
		}
		out = append(out, a)
	}
	return out, strings.Join(out, ","), nil
}

func (s *source) Token(ctx context.Context, audiences []string) (string, error) {
	normalized, key, err := normalizeAudiences(audiences)
	if err != nil {
		return "", err
	}
	if entry, ok := s.cache.Load(key); ok {
		e := entry.(*cacheEntry)
		if s.now().Add(s.skew).Before(e.exp) {
			s.observeHit(ctx, key)
			return e.token, nil
		}
	}
	s.observeMiss(ctx, key)

	v, err, _ := s.sf.Do(key, func() (any, error) {
		// re-check cache in case a prior flight finished while we queued
		if entry, ok := s.cache.Load(key); ok {
			e := entry.(*cacheEntry)
			if s.now().Add(s.skew).Before(e.exp) {
				return e.token, nil
			}
		}
		return s.fetch(ctx, normalized, key)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (s *source) fetch(ctx context.Context, normalized []string, key string) (string, error) {
	out, err := s.client.GetWebIdentityToken(ctx, &sts.GetWebIdentityTokenInput{
		Audience:         normalized,
		SigningAlgorithm: aws.String(s.algo),
		DurationSeconds:  aws.Int32(s.duration),
	})
	if err != nil {
		s.observeFetch(ctx, key, err)
		return "", fmt.Errorf("token: audience=%q: %w", key, err)
	}
	if out.WebIdentityToken == nil || out.Expiration == nil {
		s.observeFetch(ctx, key, errors.New("incomplete response"))
		return "", fmt.Errorf("token: audience=%q: STS returned incomplete response", key)
	}
	tok := *out.WebIdentityToken
	_, loaded := s.cache.Swap(key, &cacheEntry{token: tok, exp: *out.Expiration})
	if !loaded && s.inst != nil {
		s.inst.cachedSize.Add(1)
	}
	s.observeFetch(ctx, key, nil)
	return tok, nil
}

func (s *source) observeHit(ctx context.Context, key string) {
	if s.inst == nil {
		return
	}
	s.inst.CacheHit.Add(ctx, 1, metric.WithAttributes(audienceAttr(key)))
}

func (s *source) observeMiss(ctx context.Context, key string) {
	if s.inst == nil {
		return
	}
	s.inst.CacheMiss.Add(ctx, 1, metric.WithAttributes(audienceAttr(key)))
}

func (s *source) observeFetch(ctx context.Context, key string, err error) {
	if s.inst == nil {
		return
	}
	if err == nil {
		s.inst.FetchCount.Add(ctx, 1, metric.WithAttributes(audienceAttr(key), resultAttr("ok")))
		return
	}
	s.inst.FetchCount.Add(ctx, 1, metric.WithAttributes(
		audienceAttr(key),
		resultAttr("error"),
		errorClassAttr(err),
	))
}
