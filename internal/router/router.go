package router

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

var ErrNoUpstream = errors.New("no upstream resolvable")

type Resolver struct {
	pinnedHost string
	scheme     string
	headerName string
}

func New(pinnedHost, scheme, headerName string) *Resolver {
	return &Resolver{pinnedHost: pinnedHost, scheme: scheme, headerName: headerName}
}

func (r *Resolver) HeaderName() string { return r.headerName }

func (r *Resolver) Resolve(req *http.Request) (*url.URL, error) {
	host := r.pinnedHost
	if host == "" {
		host = readHostFromRequest(req, r.headerName)
	}
	if host == "" {
		return nil, ErrNoUpstream
	}
	return &url.URL{
		Scheme:   r.scheme,
		Host:     host,
		Path:     req.URL.Path,
		RawQuery: req.URL.RawQuery,
	}, nil
}

func readHostFromRequest(req *http.Request, headerName string) string {
	if strings.EqualFold(headerName, "Host") {
		return req.Host
	}
	return req.Header.Get(headerName)
}
