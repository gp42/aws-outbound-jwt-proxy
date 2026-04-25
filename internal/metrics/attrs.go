package metrics

import (
	"context"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"syscall"
)

// canonicalMethods holds the HTTP methods semconv permits as literal attribute
// values; anything else bucketizes to "_OTHER".
var canonicalMethods = map[string]struct{}{
	"GET":     {},
	"HEAD":    {},
	"POST":    {},
	"PUT":     {},
	"DELETE":  {},
	"CONNECT": {},
	"OPTIONS": {},
	"TRACE":   {},
	"PATCH":   {},
}

// CanonicalMethod returns the semconv-safe value for http.request.method.
// Empty / unknown / lowercase methods collapse to "_OTHER".
func CanonicalMethod(method string) string {
	up := strings.ToUpper(method)
	if _, ok := canonicalMethods[up]; ok {
		return up
	}
	return "_OTHER"
}

// HostAndPort splits u into host and port using scheme defaults. The emitPort
// return flag is true only when the port differs from the scheme default (80
// for http, 443 for https); callers should only attach server.port when it is.
func HostAndPort(u *url.URL) (host string, port int, emitPort bool) {
	if u == nil {
		return "", 0, false
	}
	host = u.Hostname()
	p := u.Port()
	def := defaultPort(u.Scheme)
	if p == "" {
		return host, def, false
	}
	n, err := strconv.Atoi(p)
	if err != nil {
		return host, def, false
	}
	return host, n, n != def
}

func defaultPort(scheme string) int {
	switch strings.ToLower(scheme) {
	case "https":
		return 443
	case "http":
		return 80
	default:
		return 0
	}
}

// ClassifyError maps a transport/handler error to the semconv error.type
// vocabulary this proxy uses. ok=false means no error.type attribute should
// be emitted (success case or caller decided 5xx-without-err is not an error).
func ClassifyError(err error) (string, bool) {
	if err == nil {
		return "", false
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return "timeout", true
	}
	if errors.Is(err, syscall.ECONNREFUSED) {
		return "connection_refused", true
	}
	var ne net.Error
	if errors.As(err, &ne) && ne.Timeout() {
		return "timeout", true
	}
	return "unknown", true
}

