package common

import "time"

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.0"            // this hard coding will be replaced automatically when building, no need to manually change

// IsPassthroughModel checks if the given model name is a passthrough model.
// For passthrough models, the original client Authorization header (api-key)
// and client IP (cip) are forwarded verbatim to the upstream service. The
// upstream uses these two values as load balancing identifiers, so they must
// not be covered by the channel key.
func IsPassthroughModel(modelName string) bool {
	return modelName == "bedi/deepseek-v4-flash" || modelName == "bedi/glm-4.7"
}
