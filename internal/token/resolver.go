package token

import (
	"errors"
	"net"
	"net/http"
	"strings"
)

// AudienceResolver returns the audience set to request a JWT for on a given
// outbound request. Implementations may ignore the request entirely (see
// StaticAudiences) or derive audiences from request fields (a future
// per-host resolver). The returned slice is passed verbatim to
// Source.Token; normalization (sort + dedupe) happens inside Source.
type AudienceResolver interface {
	Resolve(req *http.Request) ([]string, error)
}

// StaticAudiences returns an AudienceResolver that always yields the given
// audience slice regardless of the request. The slice is returned verbatim
// (no copy, no sort, no dedupe) — callers MUST NOT mutate it after the
// resolver is constructed.
func StaticAudiences(audiences []string) AudienceResolver {
	return staticAudiences(audiences)
}

type staticAudiences []string

func (s staticAudiences) Resolve(_ *http.Request) ([]string, error) {
	return []string(s), nil
}

// HostAudience is an AudienceResolver that derives the audience set from the
// outbound request's target URL (req.URL.Scheme + req.URL.Host, as set by the
// router). The returned slice contains a single entry of the form
// "<scheme>://<host>" where host is lowercased and any port is stripped.
type HostAudience struct{}

func (HostAudience) Resolve(req *http.Request) ([]string, error) {
	host := req.URL.Host
	if host == "" {
		return nil, errors.New("token: HostAudience resolver: request URL.Host is empty")
	}
	scheme := req.URL.Scheme
	if scheme == "" {
		return nil, errors.New("token: HostAudience resolver: request URL.Scheme is empty")
	}
	if h, _, err := net.SplitHostPort(host); err == nil {
		if strings.Contains(h, ":") {
			host = "[" + h + "]"
		} else {
			host = h
		}
	}
	return []string{strings.ToLower(scheme) + "://" + strings.ToLower(host)}, nil
}
