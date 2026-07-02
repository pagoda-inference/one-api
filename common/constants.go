package common

import "time"

var StartTime = time.Now().Unix() // unit: second
var Version = "v0.0.0"            // this hard coding will be replaced automatically when building, no need to manually change

// PassthroughModel is the model name for which the original client
// Authorization header (api-key) and client IP (cip) are forwarded verbatim
// to the upstream service. The upstream uses these two values as load
// balancing identifiers, so they must not be replaced by the channel key.
const PassthroughModel = "bedi/deepseek-v4-flash"
