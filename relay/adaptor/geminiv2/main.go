package geminiv2

import (
	"fmt"
	"strings"

	"github.com/pagoda-inference/one-api/relay/meta"
)

func GetRequestURL(meta *meta.Meta) (string, error) {
	baseURL := strings.TrimSuffix(meta.BaseURL, "/")
	requestPath := strings.TrimPrefix(meta.RequestURLPath, "/v1")
	return fmt.Sprintf("%s%s", baseURL, requestPath), nil
}
