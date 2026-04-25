// Package tokentest provides an in-memory token.Source for tests.
package tokentest

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gp42/aws-outbound-jwt-proxy/internal/token"
)

type fake struct {
	tokens map[string]string // joined-normalized audience set -> token
}

// New returns a token.Source that serves a pre-canned token per audience set.
// Map keys are comma-joined normalized audience sets (e.g. "a,b"), matching
// the form produced by the real Source. Requests for unknown sets return a
// non-nil error.
func New(tokens map[string]string) token.Source {
	m := make(map[string]string, len(tokens))
	for k, v := range tokens {
		m[normalizeKey(k)] = v
	}
	return &fake{tokens: m}
}

func (f *fake) Token(_ context.Context, audiences []string) (string, error) {
	if len(audiences) == 0 {
		return "", errors.New("tokentest: audiences must be non-empty")
	}
	key := normalize(audiences)
	if t, ok := f.tokens[key]; ok {
		return t, nil
	}
	return "", fmt.Errorf("tokentest: no token configured for audience set %q", key)
}

// normalize sorts and dedupes the slice, then joins with ",".
func normalize(audiences []string) string {
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
	return strings.Join(out, ",")
}

// normalizeKey accepts either a single audience string or a comma-joined
// list and returns it in normalized form, so map literals passed to New
// can be written loosely.
func normalizeKey(k string) string {
	parts := strings.Split(k, ",")
	return normalize(parts)
}
