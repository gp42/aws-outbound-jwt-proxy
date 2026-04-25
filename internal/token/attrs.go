package token

import (
	"errors"

	"github.com/aws/smithy-go"
	"go.opentelemetry.io/otel/attribute"
)

func audienceAttr(audience string) attribute.KeyValue {
	return attribute.String("audience", audience)
}

func resultAttr(r string) attribute.KeyValue {
	return attribute.String("result", r)
}

// errorClassAttr extracts the AWS API error code when available (e.g.
// "AccessDenied", "OutboundWebIdentityFederationDisabled"), or "transport"
// otherwise.
func errorClassAttr(err error) attribute.KeyValue {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return attribute.String("error_class", apiErr.ErrorCode())
	}
	return attribute.String("error_class", "transport")
}
