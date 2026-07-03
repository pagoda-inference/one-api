package adaptor

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/common/network"
	"github.com/pagoda-inference/one-api/relay/meta"
)

func SetupCommonRequestHeader(c *gin.Context, req *http.Request, meta *meta.Meta) {
	req.Header.Set("Content-Type", c.Request.Header.Get("Content-Type"))
	req.Header.Set("Accept", c.Request.Header.Get("Accept"))
	if meta.IsStream && c.Request.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "text/event-stream")
	}
}

func DoRequestHelper(a Adaptor, c *gin.Context, meta *meta.Meta, requestBody io.Reader) (*http.Response, error) {
	fullRequestURL, err := a.GetRequestURL(meta)
	if err != nil {
		return nil, fmt.Errorf("get request url failed: %w", err)
	}
	req, err := http.NewRequest(c.Request.Method, fullRequestURL, requestBody)
	if err != nil {
		return nil, fmt.Errorf("new request failed: %w", err)
	}
	err = a.SetupRequestHeader(c, req, meta)
	if err != nil {
		return nil, fmt.Errorf("setup request header failed: %w", err)
	}
	// For passthrough models, override the channel key with the original
	// client Authorization and forward the client IP so the upstream can
	// perform load balancing based on the real client identity.
	setupPassthroughHeaders(c, req, meta)
	logger.Debugf(c.Request.Context(), "DoRequest URL: %s, Auth: %s", fullRequestURL, req.Header.Get("Authorization"))
	resp, err := DoRequest(c, req)
	if err != nil {
		return nil, fmt.Errorf("do request failed: %w", err)
	}
	return resp, nil
}

func DoRequest(c *gin.Context, req *http.Request) (*http.Response, error) {
	resp, err := client.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, errors.New("resp is nil")
	}
	_ = req.Body.Close()
	_ = c.Request.Body.Close()
	return resp, nil
}

// setupPassthroughHeaders forwards the original client Authorization header
// (api-key) and the client IP to the upstream request. This only takes
// effect for common.PassthroughModel requests; for every other model the
// request headers built by the adaptor are left untouched.
//
// The original Authorization is captured by the middleware (see
// middleware.SetupContextForSelectedChannel) before the channel key overwrites
// the incoming Authorization header. Here we restore it on the upstream
// request so the upstream service receives the real client api-key.
//
// The client IP is forwarded using the standard X-Forwarded-For format
// (RFC 7239): the existing proxy chain is preserved and the immediate
// sender's IP is appended. The upstream can extract the original client IP
// (cip) as the leftmost entry for load balancing.
func setupPassthroughHeaders(c *gin.Context, req *http.Request, meta *meta.Meta) {
	if meta.OriginModelName != common.PassthroughModel {
		return
	}
	// Forward the original client Authorization (api-key) verbatim. Only
	// overwrite when a non-empty value was captured; otherwise the channel
	// key set by the adaptor remains in place.
	if raw, exists := c.Get(ctxkey.OriginalAuthorization); exists {
		if auth, ok := raw.(string); ok && auth != "" {
			req.Header.Set("Authorization", auth)
		}
	}
	// Forward the client IP using the standard X-Forwarded-For format:
	// preserve the existing proxy chain and append the immediate sender's
	// IP so the upstream can identify the real client (leftmost entry).
	originalXFF := c.Request.Header.Get("X-Forwarded-For")
	immediateIP := network.GetIPFromRemoteAddr(c.Request.RemoteAddr)
	switch {
	case originalXFF != "" && immediateIP != "":
		req.Header.Set("X-Forwarded-For", originalXFF+", "+immediateIP)
	case originalXFF != "":
		req.Header.Set("X-Forwarded-For", originalXFF)
	case immediateIP != "":
		req.Header.Set("X-Forwarded-For", immediateIP)
	}
}
