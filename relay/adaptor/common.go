package adaptor

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

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
	// For the passthrough upstream (http://10.1.105.193:81), forward the
	// client api-key as a separate "user-api-key" header and the client IP
	// via X-Forwarded-For so the upstream can identify the real client for
	// load balancing. The upstream Authorization stays as the channel key.
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

// setupPassthroughHeaders forwards the client api-key and client IP to the
// upstream request. This only takes effect when the request is routed to the
// passthrough upstream (common.PassthroughUpstreamBaseURL,
// http://10.1.105.193:81); for every other upstream the request headers built
// by the adaptor are left untouched. The decision is based on the upstream
// channel base URL (meta.BaseURL), not the model name.
//
// The upstream Authorization header retains the channel api-key set by the
// adaptor (i.e. normal channel authentication). The client's original
// api-key—captured by the middleware before the channel key overwrites the
// incoming Authorization—is extracted from the "Bearer <key>" format and
// forwarded as a separate "user-api-key" header so the upstream can identify
// the real client.
//
// The client IP is forwarded using the standard X-Forwarded-For format
// (RFC 7239): the existing proxy chain is preserved and the immediate
// sender's IP is appended. Loopback addresses (127.0.0.1, ::1) introduced by
// internal cluster Nginx nodes forwarding over localhost are filtered out so
// only external client IPs and legitimate non-loopback upstream IPs remain.
// The upstream can extract the original client IP (cip) as the leftmost entry
// for load balancing.
func setupPassthroughHeaders(c *gin.Context, req *http.Request, meta *meta.Meta) {
	if !common.IsPassthroughUpstream(meta.BaseURL) {
		return
	}
	// Extract the client api-key from the original Authorization header
	// (captured by middleware before the channel key overwrote it) and
	// forward it as a separate "user-api-key" header. The upstream
	// Authorization remains the channel key set by the adaptor.
	if raw, exists := c.Get(ctxkey.OriginalAuthorization); exists {
		if auth, ok := raw.(string); ok && auth != "" {
			clientKey := extractBearerToken(auth)
			if clientKey != "" {
				req.Header.Set("user-api-key", clientKey)
				logger.Debugf(c.Request.Context(), "passthrough: forwarded user-api-key for upstream %s", meta.BaseURL)
			} else {
				logger.Debugf(c.Request.Context(), "passthrough: could not extract api-key from Authorization header for upstream %s", meta.BaseURL)
			}
		}
	}
	// Forward the client IP via the standard X-Forwarded-For header
	// (RFC 7239): preserve the existing proxy chain and append the
	// immediate sender's IP, while filtering out loopback addresses
	// (127.0.0.0/8, ::1) introduced by internal cluster Nginx nodes
	// forwarding over localhost. The upstream uses the leftmost entry as
	// the original client IP (cip) for load balancing.
	if xff := buildForwardedXFF(c.Request.Header.Get("X-Forwarded-For"), c.Request.RemoteAddr); xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
}

// buildForwardedXFF constructs an X-Forwarded-For value by preserving the
// existing proxy chain and appending the immediate sender's IP (extracted
// from remoteAddr). Loopback addresses (127.0.0.0/8, ::1) — introduced by
// internal cluster Nginx nodes forwarding over localhost — are dropped so
// the upstream receives only valid, non-loopback IPs. Returns an empty
// string if no valid IP remains.
func buildForwardedXFF(existingXFF, remoteAddr string) string {
	var ips []string
	for _, p := range strings.Split(existingXFF, ",") {
		if ip := strings.TrimSpace(p); ip != "" && !network.IsLoopbackIP(ip) {
			ips = append(ips, ip)
		}
	}
	if sender := network.GetIPFromRemoteAddr(remoteAddr); sender != "" && !network.IsLoopbackIP(sender) {
		ips = append(ips, sender)
	}
	return strings.Join(ips, ", ")
}

// extractBearerToken extracts the api-key from an "Authorization: Bearer <key>"
// header value. Returns an empty string if the header is missing the "Bearer "
// prefix or the token portion is empty.
func extractBearerToken(authHeader string) string {
	const bearerPrefix = "Bearer "
	if !strings.HasPrefix(authHeader, bearerPrefix) {
		return ""
	}
	return strings.TrimSpace(authHeader[len(bearerPrefix):])
}
