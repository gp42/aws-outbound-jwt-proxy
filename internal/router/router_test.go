package router

import (
	"errors"
	"net/http/httptest"
	"testing"
)

func TestPinnedWinsOverHeader(t *testing.T) {
	r := New("api.example.com", "https", "X-Upstream-Host")
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header.Set("X-Upstream-Host", "other.example.com")
	u, err := r.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "api.example.com" || u.Scheme != "https" {
		t.Fatalf("got %v", u)
	}
}

func TestPinnedNoHeader(t *testing.T) {
	r := New("api.example.com", "https", "X-Upstream-Host")
	req := httptest.NewRequest("GET", "/foo", nil)
	u, err := r.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "api.example.com" {
		t.Fatalf("got %v", u)
	}
}

func TestDefaultHeader(t *testing.T) {
	r := New("", "https", "X-Upstream-Host")
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header.Set("X-Upstream-Host", "api.example.com")
	u, err := r.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "api.example.com" {
		t.Fatalf("got %v", u)
	}
}

func TestCustomHeaderName(t *testing.T) {
	r := New("", "https", "X-Target")
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Header.Set("X-Target", "api.example.com")
	u, err := r.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "api.example.com" {
		t.Fatalf("got %v", u)
	}
}

func TestHostHeaderSpecialCase(t *testing.T) {
	r := New("", "https", "Host")
	req := httptest.NewRequest("GET", "/foo", nil)
	req.Host = "api.example.com"
	u, err := r.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	if u.Host != "api.example.com" {
		t.Fatalf("got %v", u)
	}
}

func TestPathAndQueryPreserved(t *testing.T) {
	r := New("api.example.com", "https", "X-Upstream-Host")
	req := httptest.NewRequest("GET", "/v1/items?limit=10", nil)
	u, err := r.Resolve(req)
	if err != nil {
		t.Fatal(err)
	}
	if u.String() != "https://api.example.com/v1/items?limit=10" {
		t.Fatalf("got %s", u.String())
	}
}

func TestNoPinNoHeader(t *testing.T) {
	r := New("", "https", "X-Upstream-Host")
	req := httptest.NewRequest("GET", "/foo", nil)
	_, err := r.Resolve(req)
	if !errors.Is(err, ErrNoUpstream) {
		t.Fatalf("expected ErrNoUpstream, got %v", err)
	}
}
