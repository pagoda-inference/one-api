package adaptor

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/relay/meta"
	relaymodel "github.com/pagoda-inference/one-api/relay/model"
)

// TestExtractBearerToken verifies parsing of "Bearer <key>" Authorization headers.
func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name       string
		authHeader string
		want       string
	}{
		{"standard bearer", "Bearer sk-client-key", "sk-client-key"},
		{"bearer with trailing spaces", "Bearer sk-client-key   ", "sk-client-key"},
		{"empty token", "Bearer ", ""},
		{"no bearer prefix", "Basic dXNlcjpwYXNz", ""},
		{"empty string", "", ""},
		{"lowercase bearer not matched", "bearer sk-client-key", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractBearerToken(tt.authHeader); got != tt.want {
				t.Errorf("extractBearerToken(%q) = %q, want %q", tt.authHeader, got, tt.want)
			}
		})
	}
}

// TestSetupPassthroughHeaders verifies that for the passthrough upstream
// (http://10.1.105.193:81) the client api-key is forwarded as "user-api-key"
// while the upstream Authorization stays as the channel key. For other
// upstreams no passthrough headers are added.
func passthroughBaseURL() string {
	return common.PassthroughUpstreamBaseURL
}

func TestSetupPassthroughHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name         string
		baseURL      string
		originalAuth string
		wantAuth     string
		wantUserKey  string
	}{
		{
			name:         "passthrough upstream: channel key stays, user-api-key set",
			baseURL:      passthroughBaseURL(),
			originalAuth: "Bearer sk-client-key",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "sk-client-key",
		},
		{
			name:         "passthrough upstream with trailing slash still matches",
			baseURL:      passthroughBaseURL() + "/",
			originalAuth: "Bearer sk-client-key",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "sk-client-key",
		},
		{
			name:         "non-passthrough upstream: no user-api-key",
			baseURL:      "https://api.openai.com",
			originalAuth: "Bearer sk-client-key",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "",
		},
		{
			name:         "empty original auth: no user-api-key",
			baseURL:      passthroughBaseURL(),
			originalAuth: "",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "",
		},
		{
			name:         "non-bearer auth: no user-api-key extracted",
			baseURL:      passthroughBaseURL(),
			originalAuth: "Basic dXNlcjpwYXNz",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "",
		},
		{
			name:         "bearer with empty token: no user-api-key",
			baseURL:      passthroughBaseURL(),
			originalAuth: "Bearer ",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			c.Set(ctxkey.OriginalAuthorization, tt.originalAuth)

			req := httptest.NewRequest(http.MethodPost, "https://upstream/v1/chat/completions", nil)
			req.Header.Set("Authorization", "Bearer sk-channel-key")

			m := &meta.Meta{BaseURL: tt.baseURL}
			setupPassthroughHeaders(c, req, m)

			if got := req.Header.Get("Authorization"); got != tt.wantAuth {
				t.Errorf("Authorization = %q, want %q", got, tt.wantAuth)
			}
			if got := req.Header.Get("user-api-key"); got != tt.wantUserKey {
				t.Errorf("user-api-key = %q, want %q", got, tt.wantUserKey)
			}
		})
	}
}

// stubAdaptor implements the Adaptor interface for integration testing of
// DoRequestHelper. Its SetupRequestHeader sets a channel key (as a real
// adaptor would), which setupPassthroughHeaders must NOT override.
type stubAdaptor struct {
	upstreamURL string
}

func (a *stubAdaptor) Init(_ *meta.Meta)                          {}
func (a *stubAdaptor) GetRequestURL(_ *meta.Meta) (string, error) { return a.upstreamURL, nil }
func (a *stubAdaptor) SetupRequestHeader(_ *gin.Context, req *http.Request, _ *meta.Meta) error {
	req.Header.Set("Authorization", "Bearer sk-channel-key")
	req.Header.Set("Content-Type", "application/json")
	return nil
}
func (a *stubAdaptor) ConvertRequest(_ *gin.Context, _ int, r *relaymodel.GeneralOpenAIRequest) (any, error) {
	return r, nil
}
func (a *stubAdaptor) ConvertImageRequest(r *relaymodel.ImageRequest) (any, error) {
	return r, nil
}
func (a *stubAdaptor) DoRequest(c *gin.Context, m *meta.Meta, body io.Reader) (*http.Response, error) {
	return DoRequestHelper(a, c, m, body)
}
func (a *stubAdaptor) DoResponse(_ *gin.Context, _ *http.Response, _ *meta.Meta) (*relaymodel.Usage, *relaymodel.ErrorWithStatusCode) {
	return &relaymodel.Usage{}, nil
}
func (a *stubAdaptor) GetModelList() []string { return nil }
func (a *stubAdaptor) GetChannelName() string { return "stub" }

// TestDoRequestHelperPassthroughIntegration is an integration test that spins
// up a fake upstream server and verifies that DoRequestHelper forwards the
// client api-key as "user-api-key" while keeping the channel key in
// Authorization when the channel base URL is the passthrough upstream.
// The stub adaptor always sends the HTTP request to the test server, while
// meta.BaseURL controls whether the passthrough logic is triggered.
func TestDoRequestHelperPassthroughIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	if client.HTTPClient == nil {
		client.HTTPClient = &http.Client{}
	}

	type upstreamHeaders struct {
		auth    string
		userKey string
	}
	headerCh := make(chan upstreamHeaders, 1)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerCh <- upstreamHeaders{
			auth:    r.Header.Get("Authorization"),
			userKey: r.Header.Get("user-api-key"),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("{\"choices\":[]}"))
	}))
	defer srv.Close()

	tests := []struct {
		name         string
		baseURL      string
		originalAuth string
		wantAuth     string
		wantUserKey  string
	}{
		{
			name:         "passthrough upstream: upstream receives channel key + user-api-key",
			baseURL:      passthroughBaseURL(),
			originalAuth: "Bearer sk-client-key",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "sk-client-key",
		},
		{
			name:         "other upstream: upstream receives channel key only, no user-api-key",
			baseURL:      "https://api.openai.com",
			originalAuth: "Bearer sk-client-key",
			wantAuth:     "Bearer sk-channel-key",
			wantUserKey:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _ := gin.CreateTestContext(httptest.NewRecorder())
			c.Request = httptest.NewRequest(http.MethodPost, "/v1/chat/completions", strings.NewReader(""))
			c.Set(ctxkey.OriginalAuthorization, tt.originalAuth)

			m := &meta.Meta{BaseURL: tt.baseURL}
			a := &stubAdaptor{upstreamURL: srv.URL}

			resp, err := DoRequestHelper(a, c, m, strings.NewReader(""))
			if err != nil {
				t.Fatalf("DoRequestHelper failed: %v", err)
			}
			defer resp.Body.Close()

			got := <-headerCh
			if got.auth != tt.wantAuth {
				t.Errorf("upstream Authorization = %q, want %q", got.auth, tt.wantAuth)
			}
			if got.userKey != tt.wantUserKey {
				t.Errorf("upstream user-api-key = %q, want %q", got.userKey, tt.wantUserKey)
			}
		})
	}
}
