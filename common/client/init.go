package client

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/logger"
)

var HTTPClient *http.Client
var ImpatientHTTPClient *http.Client
var UserContentRequestHTTPClient *http.Client

// Default connection pool settings optimized for high concurrency
const (
	DefaultMaxIdleConns          = 1000
	DefaultMaxIdleConnsPerHost   = 100
	DefaultMaxConnsPerHost       = 200
	DefaultIdleConnTimeout       = 90 * time.Second
	DefaultTLSHandshakeTimeout   = 10 * time.Second
	DefaultResponseHeaderTimeout = 10 * time.Second
	DefaultExpectContinueTimeout = 1 * time.Second
	DefaultDialTimeout           = 30 * time.Second
	DefaultKeepAlive             = 30 * time.Second
)

// createTransport creates an optimized HTTP transport with connection pooling
func createTransport(proxyURL string) *http.Transport {
	transport := &http.Transport{
		// Connection pool settings
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxIdleConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       DefaultIdleConnTimeout,

		// Timeout settings
		TLSHandshakeTimeout:   DefaultTLSHandshakeTimeout,
		ResponseHeaderTimeout: DefaultResponseHeaderTimeout,
		ExpectContinueTimeout: DefaultExpectContinueTimeout,

		// Dial settings
		DialContext: (&net.Dialer{
			Timeout:   DefaultDialTimeout,
			KeepAlive: DefaultKeepAlive,
		}).DialContext,

		// Enable HTTP/2
		ForceAttemptHTTP2: true,

		// TLS settings
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: false,
		},
	}

	// Set proxy if configured
	if proxyURL != "" {
		proxy, err := url.Parse(proxyURL)
		if err != nil {
			logger.SysError(fmt.Sprintf("failed to parse proxy URL: %s", err.Error()))
		} else {
			transport.Proxy = http.ProxyURL(proxy)
		}
	}

	return transport
}

func Init() {
	// Initialize UserContentRequestHTTPClient
	userContentTransport := createTransport(config.UserContentRequestProxy)
	UserContentRequestHTTPClient = &http.Client{
		Transport: userContentTransport,
		Timeout:   time.Second * time.Duration(config.UserContentRequestTimeout),
	}
	if config.UserContentRequestProxy != "" {
		logger.SysLog(fmt.Sprintf("using %s as proxy to fetch user content", config.UserContentRequestProxy))
	}

	// Initialize HTTPClient for relay
	relayTransport := createTransport(config.RelayProxy)
	if config.RelayProxy != "" {
		logger.SysLog(fmt.Sprintf("using %s as api relay proxy", config.RelayProxy))
	}

	if config.RelayTimeout == 0 {
		HTTPClient = &http.Client{
			Transport: relayTransport,
		}
	} else {
		HTTPClient = &http.Client{
			Timeout:   time.Duration(config.RelayTimeout) * time.Second,
			Transport: relayTransport,
		}
	}

	// Initialize ImpatientHTTPClient for quick operations
	ImpatientHTTPClient = &http.Client{
		Timeout:   5 * time.Second,
		Transport: relayTransport,
	}

	// Log connection pool settings
	logger.SysLog(fmt.Sprintf("HTTP client initialized: MaxIdleConns=%d, MaxIdleConnsPerHost=%d, MaxConnsPerHost=%d",
		config.MaxIdleConns, config.MaxIdleConnsPerHost, config.MaxConnsPerHost))
}