package tokentest

import (
	"context"
	"testing"
)

func TestKnownAudience(t *testing.T) {
	src := New(map[string]string{"A": "token-a"})
	got, err := src.Token(context.Background(), []string{"A"})
	if err != nil {
		t.Fatal(err)
	}
	if got != "token-a" {
		t.Fatalf("got %q", got)
	}
}

func TestUnknownAudience(t *testing.T) {
	src := New(map[string]string{"A": "token-a"})
	if _, err := src.Token(context.Background(), []string{"missing"}); err == nil {
		t.Fatal("expected error")
	}
}

func TestMultiAudienceOrderInsensitive(t *testing.T) {
	src := New(map[string]string{"a,b": "token-ab"})
	for _, in := range [][]string{{"a", "b"}, {"b", "a"}, {"a", "b", "a"}} {
		got, err := src.Token(context.Background(), in)
		if err != nil {
			t.Fatalf("%v: %v", in, err)
		}
		if got != "token-ab" {
			t.Fatalf("%v: got %q", in, got)
		}
	}
}

func TestEmptyAudienceRejected(t *testing.T) {
	src := New(map[string]string{"A": "t"})
	if _, err := src.Token(context.Background(), nil); err == nil {
		t.Fatal("expected error for nil audiences")
	}
	if _, err := src.Token(context.Background(), []string{}); err == nil {
		t.Fatal("expected error for empty audiences")
	}
}
