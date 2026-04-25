package token

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/aws/smithy-go"
	"github.com/gp42/aws-outbound-jwt-proxy/internal/config"
)

type stubSTS struct {
	mu      sync.Mutex
	calls   int
	handler func(in *sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error)
}

func (s *stubSTS) GetWebIdentityToken(ctx context.Context, in *sts.GetWebIdentityTokenInput, optFns ...func(*sts.Options)) (*sts.GetWebIdentityTokenOutput, error) {
	s.mu.Lock()
	s.calls++
	s.mu.Unlock()
	if s.handler != nil {
		return s.handler(in)
	}
	return &sts.GetWebIdentityTokenOutput{
		WebIdentityToken: aws.String("tok-" + in.Audience[0]),
		Expiration:       aws.Time(time.Now().Add(1 * time.Hour)),
	}, nil
}

func (s *stubSTS) callCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.calls
}

func newTestSource(t *testing.T, stub *stubSTS, now func() time.Time) *source {
	t.Helper()
	cfg := &config.Config{
		TokenSigningAlgorithm: "RS256",
		TokenDuration:         15 * time.Minute,
		TokenRefreshSkew:      60 * time.Second,
	}
	return newWithClient(cfg, stub, nil, now)
}

func TestCacheMissCallsSTS(t *testing.T) {
	var got *sts.GetWebIdentityTokenInput
	stub := &stubSTS{handler: func(in *sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		got = in
		return &sts.GetWebIdentityTokenOutput{
			WebIdentityToken: aws.String("abc"),
			Expiration:       aws.Time(time.Now().Add(time.Hour)),
		}, nil
	}}
	s := newTestSource(t, stub, time.Now)
	tok, err := s.Token(context.Background(), []string{"https://api.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if tok != "abc" {
		t.Fatalf("got %q", tok)
	}
	if got == nil || len(got.Audience) != 1 || got.Audience[0] != "https://api.example.com" {
		t.Fatalf("audience wrong: %+v", got)
	}
	if got.SigningAlgorithm == nil || *got.SigningAlgorithm != "RS256" {
		t.Fatalf("algo wrong: %+v", got.SigningAlgorithm)
	}
	if got.DurationSeconds == nil || *got.DurationSeconds != 900 {
		t.Fatalf("duration wrong: %+v", got.DurationSeconds)
	}
}

func TestCacheMissWithMultipleAudiences(t *testing.T) {
	var got *sts.GetWebIdentityTokenInput
	stub := &stubSTS{handler: func(in *sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		got = in
		return &sts.GetWebIdentityTokenOutput{
			WebIdentityToken: aws.String("multi"),
			Expiration:       aws.Time(time.Now().Add(time.Hour)),
		}, nil
	}}
	s := newTestSource(t, stub, time.Now)
	tok, err := s.Token(context.Background(), []string{"https://a.example.com", "https://b.example.com"})
	if err != nil {
		t.Fatal(err)
	}
	if tok != "multi" {
		t.Fatalf("got %q", tok)
	}
	if !reflect.DeepEqual(got.Audience, []string{"https://a.example.com", "https://b.example.com"}) {
		t.Fatalf("audience slice wrong: %+v", got.Audience)
	}
}

func TestCacheOrderAndDedupeNormalized(t *testing.T) {
	stub := &stubSTS{}
	s := newTestSource(t, stub, time.Now)
	// Different orderings + duplicates should all hit a single cache entry
	// and result in exactly one STS call.
	for _, in := range [][]string{{"b", "a"}, {"a", "b"}, {"a", "b", "a"}, {"b", "a", "b"}} {
		if _, err := s.Token(context.Background(), in); err != nil {
			t.Fatalf("%v: %v", in, err)
		}
	}
	if stub.callCount() != 1 {
		t.Fatalf("expected 1 STS call across normalized sets, got %d", stub.callCount())
	}
}

func TestSupersetDoesNotShareCache(t *testing.T) {
	stub := &stubSTS{}
	s := newTestSource(t, stub, time.Now)
	if _, err := s.Token(context.Background(), []string{"A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Token(context.Background(), []string{"A", "B"}); err != nil {
		t.Fatal(err)
	}
	if stub.callCount() != 2 {
		t.Fatalf("expected 2 STS calls, got %d", stub.callCount())
	}
}

func TestEmptyAudiencesRejected(t *testing.T) {
	stub := &stubSTS{}
	s := newTestSource(t, stub, time.Now)
	if _, err := s.Token(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil audiences")
	}
	if _, err := s.Token(context.Background(), []string{}); err == nil {
		t.Fatal("expected error for empty audiences")
	}
	if stub.callCount() != 0 {
		t.Fatalf("STS must not be called on empty audiences, got %d", stub.callCount())
	}
}

func TestCacheHitSkipsSTS(t *testing.T) {
	stub := &stubSTS{}
	s := newTestSource(t, stub, time.Now)
	for i := 0; i < 5; i++ {
		if _, err := s.Token(context.Background(), []string{"A"}); err != nil {
			t.Fatal(err)
		}
	}
	if stub.callCount() != 1 {
		t.Fatalf("calls: %d", stub.callCount())
	}
}

func TestNearExpiryRefresh(t *testing.T) {
	var issued int32
	stub := &stubSTS{handler: func(in *sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		n := atomic.AddInt32(&issued, 1)
		return &sts.GetWebIdentityTokenOutput{
			WebIdentityToken: aws.String(fmt.Sprintf("tok-%d", n)),
			// exp is 30s from virtual now; skew is 60s -> treated stale.
			Expiration: aws.Time(time.Unix(int64(1000+30), 0)),
		}, nil
	}}
	now := func() time.Time { return time.Unix(1000, 0) }
	s := newTestSource(t, stub, now)
	t1, err := s.Token(context.Background(), []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	t2, err := s.Token(context.Background(), []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	if t1 == t2 {
		t.Fatalf("expected refresh, got same token %q twice", t1)
	}
	if stub.callCount() != 2 {
		t.Fatalf("calls: %d", stub.callCount())
	}
}

func TestDistinctAudiencesIndependent(t *testing.T) {
	stub := &stubSTS{}
	s := newTestSource(t, stub, time.Now)
	a, err := s.Token(context.Background(), []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Token(context.Background(), []string{"B"})
	if err != nil {
		t.Fatal(err)
	}
	if a == b {
		t.Fatalf("expected distinct tokens")
	}
	if stub.callCount() != 2 {
		t.Fatalf("calls: %d", stub.callCount())
	}
	// subsequent calls hit cache
	if _, err := s.Token(context.Background(), []string{"A"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Token(context.Background(), []string{"B"}); err != nil {
		t.Fatal(err)
	}
	if stub.callCount() != 2 {
		t.Fatalf("calls: %d", stub.callCount())
	}
}

func TestSingleFlight(t *testing.T) {
	gate := make(chan struct{})
	stub := &stubSTS{handler: func(in *sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		<-gate
		return &sts.GetWebIdentityTokenOutput{
			WebIdentityToken: aws.String("coalesced"),
			Expiration:       aws.Time(time.Now().Add(time.Hour)),
		}, nil
	}}
	s := newTestSource(t, stub, time.Now)

	var wg sync.WaitGroup
	tokens := make([]string, 100)
	errs := make([]error, 100)
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			tokens[i], errs[i] = s.Token(context.Background(), []string{"A"})
		}(i)
	}
	// give goroutines a moment to all enter singleflight
	time.Sleep(50 * time.Millisecond)
	close(gate)
	wg.Wait()
	for i := 0; i < 100; i++ {
		if errs[i] != nil {
			t.Fatalf("goroutine %d: %v", i, errs[i])
		}
		if tokens[i] != "coalesced" {
			t.Fatalf("goroutine %d: %q", i, tokens[i])
		}
	}
	if stub.callCount() != 1 {
		t.Fatalf("expected 1 STS call, got %d", stub.callCount())
	}
}

func TestSingleFlightCoalescesDifferentOrderings(t *testing.T) {
	gate := make(chan struct{})
	stub := &stubSTS{handler: func(in *sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		<-gate
		return &sts.GetWebIdentityTokenOutput{
			WebIdentityToken: aws.String("shared"),
			Expiration:       aws.Time(time.Now().Add(time.Hour)),
		}, nil
	}}
	s := newTestSource(t, stub, time.Now)

	var wg sync.WaitGroup
	for _, in := range [][]string{{"a", "b"}, {"b", "a"}, {"a", "b"}, {"b", "a"}} {
		in := in
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Token(context.Background(), in); err != nil {
				t.Errorf("err: %v", err)
			}
		}()
	}
	time.Sleep(50 * time.Millisecond)
	close(gate)
	wg.Wait()
	if stub.callCount() != 1 {
		t.Fatalf("expected 1 STS call across orderings, got %d", stub.callCount())
	}
}

type apiErr struct {
	code string
	msg  string
}

func (e *apiErr) Error() string                 { return e.msg }
func (e *apiErr) ErrorCode() string             { return e.code }
func (e *apiErr) ErrorMessage() string          { return e.msg }
func (e *apiErr) ErrorFault() smithy.ErrorFault { return smithy.FaultClient }

func TestAccessDeniedPreserved(t *testing.T) {
	stub := &stubSTS{handler: func(*sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		return nil, &apiErr{code: "AccessDenied", msg: "not allowed"}
	}}
	s := newTestSource(t, stub, time.Now)
	_, err := s.Token(context.Background(), []string{"A"})
	if err == nil {
		t.Fatal("expected error")
	}
	var api smithy.APIError
	if !errors.As(err, &api) {
		t.Fatalf("error not unwrappable to APIError: %v", err)
	}
	if api.ErrorCode() != "AccessDenied" {
		t.Fatalf("code: %q", api.ErrorCode())
	}
}

func TestOutboundWebIdentityFederationDisabledPreserved(t *testing.T) {
	stub := &stubSTS{handler: func(*sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		return nil, &apiErr{code: "OutboundWebIdentityFederationDisabled", msg: "disabled"}
	}}
	s := newTestSource(t, stub, time.Now)
	_, err := s.Token(context.Background(), []string{"A"})
	if err == nil {
		t.Fatal("expected error")
	}
	var api smithy.APIError
	if !errors.As(err, &api) || api.ErrorCode() != "OutboundWebIdentityFederationDisabled" {
		t.Fatalf("unexpected: %v", err)
	}
}

func TestTransportErrorNotCached(t *testing.T) {
	failing := errors.New("dial tcp: connection refused")
	stub := &stubSTS{handler: func(*sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		return nil, failing
	}}
	s := newTestSource(t, stub, time.Now)
	if _, err := s.Token(context.Background(), []string{"A"}); err == nil {
		t.Fatal("expected error")
	}
	if _, ok := s.cache.Load("A"); ok {
		t.Fatal("cache populated on error")
	}
	// second call retries
	if _, err := s.Token(context.Background(), []string{"A"}); err == nil {
		t.Fatal("expected error")
	}
	if stub.callCount() != 2 {
		t.Fatalf("calls: %d", stub.callCount())
	}
}

func TestContextCancellation(t *testing.T) {
	stub := &stubSTS{handler: func(*sts.GetWebIdentityTokenInput) (*sts.GetWebIdentityTokenOutput, error) {
		return nil, context.Canceled
	}}
	s := newTestSource(t, stub, time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := s.Token(ctx, []string{"A"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if _, ok := s.cache.Load("A"); ok {
		t.Fatal("cache populated on cancel")
	}
}
