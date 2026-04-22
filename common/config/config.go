package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pagoda-inference/one-api/common/env"

	"github.com/google/uuid"
)

var SystemName = "One API"
var ServerAddress = env.String("SERVER_ADDRESS", "https://baotaai.bedicloud.net")
var Footer = ""
var Logo = ""
var TopUpLink = ""
var ChatLink = ""
var QuotaPerUnit = 500 * 1000.0 // $0.002 / 1K tokens
var DisplayInCurrencyEnabled = true
var DisplayTokenStatEnabled = true

// Any options with "Secret", "Token" in its key won't be return by GetOptions

var SessionSecret = uuid.New().String()

var OptionMap map[string]string
var OptionMapRWMutex sync.RWMutex

var ItemsPerPage = 10
var MaxRecentItems = 100

var PasswordLoginEnabled = true
var PasswordRegisterEnabled = true
var EmailVerificationEnabled = false
var GitHubOAuthEnabled = false
var OidcEnabled = false
var WeChatAuthEnabled = false
var TurnstileCheckEnabled = false
var RegisterEnabled = true

var EmailDomainRestrictionEnabled = false
var EmailDomainWhitelist = []string{
	"gmail.com",
	"163.com",
	"126.com",
	"qq.com",
	"outlook.com",
	"hotmail.com",
	"icloud.com",
	"yahoo.com",
	"foxmail.com",
}

var DebugEnabled = strings.ToLower(os.Getenv("DEBUG")) == "true"
var DebugSQLEnabled = strings.ToLower(os.Getenv("DEBUG_SQL")) == "true"
var MemoryCacheEnabled = strings.ToLower(os.Getenv("MEMORY_CACHE_ENABLED")) == "true"

var LogConsumeEnabled = true

var SMTPServer = ""
var SMTPPort = 587
var SMTPAccount = ""
var SMTPFrom = ""
var SMTPToken = ""

var GitHubClientId = env.String("GitHubClientId", "")
var GitHubClientSecret = env.String("GitHubClientSecret", "")

var LarkClientId = env.String("LarkClientId", "")
var LarkClientSecret = env.String("LarkClientSecret", "")

var OidcClientId = ""
var OidcClientSecret = ""
var OidcWellKnown = ""
var OidcAuthorizationEndpoint = ""
var OidcTokenEndpoint = ""
var OidcUserinfoEndpoint = ""

var WeChatServerAddress = ""
var WeChatServerToken = ""
var WeChatAccountQRCodeImageURL = ""

var MessagePusherAddress = ""
var MessagePusherToken = ""

var TurnstileSiteKey = ""
var TurnstileSecretKey = ""

var QuotaForNewUser int64 = 0
var QuotaForInviter int64 = 0
var QuotaForInvitee int64 = 0
var ChannelDisableThreshold = 5.0
var AutomaticDisableChannelEnabled = false
var AutomaticEnableChannelEnabled = false
var QuotaRemindThreshold int64 = 1000
var PreConsumedQuota int64 = 500
var ApproximateTokenEnabled = false
var RetryTimes = 0

var RootUserEmail = ""

var IsMasterNode = os.Getenv("NODE_TYPE") != "slave"

var requestInterval, _ = strconv.Atoi(os.Getenv("POLLING_INTERVAL"))
var RequestInterval = time.Duration(requestInterval) * time.Second

var SyncFrequency = env.Int("SYNC_FREQUENCY", 10*60) // unit is second

var BatchUpdateEnabled = false
var BatchUpdateInterval = env.Int("BATCH_UPDATE_INTERVAL", 5)

var RelayTimeout = env.Int("RELAY_TIMEOUT", 0) // unit is second

var ResponseHeaderTimeout = env.Int("RESPONSE_HEADER_TIMEOUT", 60) // unit is second

var GeminiSafetySetting = env.String("GEMINI_SAFETY_SETTING", "BLOCK_NONE")

var Theme = env.String("THEME", "default")
var ValidThemes = map[string]bool{
	"default": true,
	"berry":   true,
	"air":     true,
}

// All duration's unit is seconds
// Shouldn't larger then RateLimitKeyExpirationDuration
var (
	GlobalApiRateLimitNum            = env.Int("GLOBAL_API_RATE_LIMIT", 480)
	GlobalApiRateLimitDuration int64 = 3 * 60

	GlobalWebRateLimitNum            = env.Int("GLOBAL_WEB_RATE_LIMIT", 240)
	GlobalWebRateLimitDuration int64 = 3 * 60

	UploadRateLimitNum            = 10
	UploadRateLimitDuration int64 = 60

	DownloadRateLimitNum            = 10
	DownloadRateLimitDuration int64 = 60

	CriticalRateLimitNum            = 20
	CriticalRateLimitDuration int64 = 20 * 60
)

var RateLimitKeyExpirationDuration = 20 * time.Minute

var EnableMetric = env.Bool("ENABLE_METRIC", false)
var MetricQueueSize = env.Int("METRIC_QUEUE_SIZE", 10)
var MetricSuccessRateThreshold = env.Float64("METRIC_SUCCESS_RATE_THRESHOLD", 0.8)
var MetricSuccessChanSize = env.Int("METRIC_SUCCESS_CHAN_SIZE", 1024)
var MetricFailChanSize = env.Int("METRIC_FAIL_CHAN_SIZE", 128)

var LogRetentionDays = env.Int("LOG_RETENTION_DAYS", 90) // unit is days, 0 means no cleanup

var InitialRootToken = os.Getenv("INITIAL_ROOT_TOKEN")

var InitialRootAccessToken = os.Getenv("INITIAL_ROOT_ACCESS_TOKEN")

var GeminiVersion = env.String("GEMINI_VERSION", "v1")

var OnlyOneLogFile = env.Bool("ONLY_ONE_LOG_FILE", false)

var RelayProxy = env.String("RELAY_PROXY", "")
var UserContentRequestProxy = env.String("USER_CONTENT_REQUEST_PROXY", "")
var UserContentRequestTimeout = env.Int("USER_CONTENT_REQUEST_TIMEOUT", 30)

var EnforceIncludeUsage = env.Bool("ENFORCE_INCLUDE_USAGE", false)
var TestPrompt = env.String("TEST_PROMPT", "Output only your specific model name with no additional text.")

// HTTP Connection Pool Settings
var MaxIdleConns = env.Int("MAX_IDLE_CONNS", 1000)
var MaxIdleConnsPerHost = env.Int("MAX_IDLE_CONNS_PER_HOST", 100)

// MinIO / S3 Object Storage Settings
var MinIOEnabled = env.Bool("MINIO_ENABLED", false)
var MinIOEndpoint = env.String("MINIO_ENDPOINT", "baota-oneapi-s3-svc:9000")
var MinIOAccessKey = env.String("MINIO_ACCESS_KEY", "minio")
var MinIOSecretKey = env.String("MINIO_SECRET_KEY", "Baota1234.com")
var MinIOBucket = env.String("MINIO_BUCKET", "minio-test")
var MinIOPublicBaseURL = env.String("MINIO_PUBLIC_BASE_URL", "http://baota-oneapi-s3-svc:9000/minio-test")
var MinIOUseSSL = env.Bool("MINIO_USE_SSL", false)
var MaxConnsPerHost = env.Int("MAX_CONNS_PER_HOST", 200)

// Health Check Settings
var HealthCheckInterval = env.Int("HEALTH_CHECK_INTERVAL", 30)
var HealthCheckFailThreshold = env.Int("HEALTH_CHECK_FAIL_THRESHOLD", 3)

// Circuit Breaker Settings
var CircuitBreakerThreshold = env.Int("CIRCUIT_BREAKER_THRESHOLD", 5)
var CircuitBreakerSuccessThreshold = env.Int("CIRCUIT_BREAKER_SUCCESS_THRESHOLD", 3)
var CircuitBreakerTimeout = env.Int("CIRCUIT_BREAKER_TIMEOUT", 30)

// Request Queue Settings
var MaxConcurrentRequests = env.Int("MAX_CONCURRENT_REQUESTS", 1000)
var RequestQueueTimeout = env.Int("REQUEST_QUEUE_TIMEOUT", 5)

// Load Balancing Settings
var EnableWeightedLoadBalancing = env.Bool("ENABLE_WEIGHTED_LB", true)

// Metrics Settings
var MetricsEnabled = env.Bool("METRICS_ENABLED", true)
var MetricsPath = env.String("METRICS_PATH", "/metrics")

// Least Connection Load Balancing Settings
var EnableLeastConnectionLB = env.Bool("ENABLE_LEAST_CONNECTION_LB", false)

// Payment Settings
var PaymentEnabled = env.Bool("PAYMENT_ENABLED", true)
var PaymentExchangeRate = env.Float64("PAYMENT_EXCHANGE_RATE", 7200) // 1元 = 7200 quota (≈ 1 USD)
var PaymentOrderExpiry = env.Int("PAYMENT_ORDER_EXPIRY", 30) // minutes

// Alipay Settings
var AlipayEnabled = env.Bool("ALIPAY_ENABLED", false)
var AlipayAppId = env.String("ALIPAY_APP_ID", "")
var AlipayPrivateKey = env.String("ALIPAY_PRIVATE_KEY", "")
var AlipayPublicKey = env.String("ALIPAY_PUBLIC_KEY", "")
var AlipayNotifyUrl = env.String("ALIPAY_NOTIFY_URL", "")

// Wechat Pay Settings
var WechatPayEnabled = env.Bool("WECHATPAY_ENABLED", false)
var WechatPayAppId = env.String("WECHATPAY_APP_ID", "")
var WechatPayMchId = env.String("WECHATPAY_MCH_ID", "")
var WechatPayApiKey = env.String("WECHATPAY_API_KEY", "")
var WechatPayNotifyUrl = env.String("WECHATPAY_NOTIFY_URL", "")
