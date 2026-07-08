package common

import (
	"strings"
	"time"
)

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.0"            // this hard coding will be replaced automatically when building, no need to manually change

// PassthroughUpstreamBaseURL is the upstream channel base URL for which the
// client api-key and client IP must be forwarded to support upstream load
// balancing. Requests routed to this upstream are treated as passthrough
// requests regardless of the requested model name.
const PassthroughUpstreamBaseURL = "http://10.1.105.193:81"

// IsPassthroughUpstream checks whether the given upstream channel base URL is
// the passthrough upstream (PassthroughUpstreamBaseURL). For requests routed
// to this upstream, the original client Authorization header (api-key) and
// client IP (cip) are forwarded to the upstream service. The upstream uses
// these two values as load balancing identifiers, so they must not be covered
// by the channel key.
//
// The comparison normalizes trailing slashes so that a channel configured with
// or without a trailing slash is treated equivalently.
func IsPassthroughUpstream(baseURL string) bool {
	return strings.TrimRight(baseURL, "/") == strings.TrimRight(PassthroughUpstreamBaseURL, "/")
}
