package token

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestStaticAudiencesReturnsSliceVerbatim(t *testing.T) {
	in := []string{"b", "a"}
	r := StaticAudiences(in)
	got, err := r.Resolve(httptest.NewRequest("GET", "/x", nil))
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if !reflect.DeepEqual(got, []string{"b", "a"}) {
		t.Fatalf("resolver must not reorder: got %v", got)
	}
}

func TestStaticAudiencesIgnoresRequest(t *testing.T) {
	r := StaticAudiences([]string{"a"})
	a, _ := r.Resolve(httptest.NewRequest("GET", "/x", nil))
	b, _ := r.Resolve(httptest.NewRequest("POST", "/y?z=1", nil))
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("resolver should return same slice regardless of request: %v vs %v", a, b)
	}
}

func TestHostAudienceNormalizesHost(t *testing.T) {
	cases := []struct {
		name   string
		scheme string
		host   string
		want   []string
	}{
		{"host with port", "https", "API.example.com:443", []string{"https://api.example.com"}},
		{"host without port", "https", "Api.Example.COM", []string{"https://api.example.com"}},
		{"ipv6 with port", "https", "[2001:db8::1]:8443", []string{"https://[2001:db8::1]"}},
		{"ipv6 without port", "https", "[2001:db8::1]", []string{"https://[2001:db8::1]"}},
		{"http scheme preserved", "http", "api.example.com", []string{"http://api.example.com"}},
		{"mixed case scheme lowercased", "HTTPS", "api.example.com", []string{"https://api.example.com"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/x", nil)
			req.URL.Scheme = tc.scheme
			req.URL.Host = tc.host
			got, err := HostAudience{}.Resolve(req)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestHostAudienceEmptyHostErrors(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	req.URL.Host = ""
	req.URL.Scheme = "https"
	_, err := HostAudience{}.Resolve(req)
	if err == nil {
		t.Fatal("expected error for empty host")
	}
}

func TestHostAudienceEmptySchemeErrors(t *testing.T) {
	req := httptest.NewRequest("GET", "/x", nil)
	req.URL.Host = "api.example.com"
	req.URL.Scheme = ""
	_, err := HostAudience{}.Resolve(req)
	if err == nil {
		t.Fatal("expected error for empty scheme")
	}
}
