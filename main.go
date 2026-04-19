package main

import (
	"embed"
	"fmt"
	"os"
	"strconv"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	_ "github.com/joho/godotenv/autoload"

	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/client"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/i18n"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/controller"
	"github.com/pagoda-inference/one-api/middleware"
	"github.com/pagoda-inference/one-api/model"
	"github.com/pagoda-inference/one-api/monitor"
	"github.com/pagoda-inference/one-api/relay/adaptor/openai"
	"github.com/pagoda-inference/one-api/router"
)

//go:embed web/build/*
var buildFS embed.FS

func main() {
	common.Init()
	logger.SetupLogger()
	logger.SysLogf("One API %s started", common.Version)

	if os.Getenv("GIN_MODE") != gin.DebugMode {
		gin.SetMode(gin.ReleaseMode)
	}
	if config.DebugEnabled {
		logger.SysLog("running in debug mode")
	}

	// Initialize SQL Database
	model.InitDB()
	model.InitLogDB()

	var err error
	err = model.CreateRootAccountIfNeed()
	if err != nil {
		logger.FatalLog("database init error: " + err.Error())
	}
	defer func() {
		err := model.CloseDB()
		if err != nil {
			logger.FatalLog("failed to close database: " + err.Error())
		}
	}()

	// Initialize Redis
	err = common.InitRedisClient()
	if err != nil {
		logger.FatalLog("failed to initialize Redis: " + err.Error())
	}

	// Initialize options
	model.InitOptionMap()
	logger.SysLog(fmt.Sprintf("using theme %s", config.Theme))
	if common.RedisEnabled {
		// for compatibility with old versions
		config.MemoryCacheEnabled = true
	}
	if config.MemoryCacheEnabled {
		logger.SysLog("memory cache enabled")
		logger.SysLog(fmt.Sprintf("sync frequency: %d seconds", config.SyncFrequency))
		model.InitChannelCache()
	}
	if config.MemoryCacheEnabled {
		go model.SyncOptions(config.SyncFrequency)
		go model.SyncChannelCache(config.SyncFrequency)
	}
	if os.Getenv("CHANNEL_TEST_FREQUENCY") != "" {
		frequency, err := strconv.Atoi(os.Getenv("CHANNEL_TEST_FREQUENCY"))
		if err != nil {
			logger.FatalLog("failed to parse CHANNEL_TEST_FREQUENCY: " + err.Error())
		}
		go controller.AutomaticallyTestChannels(frequency)
	}
	if os.Getenv("BATCH_UPDATE_ENABLED") == "true" {
		config.BatchUpdateEnabled = true
		logger.SysLog("batch update enabled with interval " + strconv.Itoa(config.BatchUpdateInterval) + "s")
		model.InitBatchUpdater()
	}
	if config.EnableMetric {
		logger.SysLog("metric enabled, will disable channel if too much request failed")
	}
	openai.InitTokenEncoders()
	client.Init()

	// Start health checker for proactive channel monitoring
	monitor.StartHealthChecker()
	if config.HealthCheckInterval > 0 {
		logger.SysLog(fmt.Sprintf("health checker enabled with interval: %d seconds", config.HealthCheckInterval))
	}

	// Initialize request queue for backpressure control
	middleware.InitRequestQueue()
	logger.SysLog(fmt.Sprintf("request queue enabled: max_concurrent=%d, timeout=%ds",
		config.MaxConcurrentRequests, config.RequestQueueTimeout))

	// Start payment cleanup worker
	controller.StartPaymentCleanupWorker()
	// Start log cleanup worker
	controller.StartLogCleanupWorker()
	if config.PaymentEnabled {
		logger.SysLog(fmt.Sprintf("payment enabled: exchange_rate=%.2f", config.PaymentExchangeRate))
	}

	// Initialize default models for marketplace
	if err := model.InitializeDefaultModels(); err != nil {
		logger.SysErrorf("failed to initialize default models: " + err.Error())
	} else {
		logger.SysLog("model marketplace initialized with default models")
	}

	// Initialize default providers
	if err := model.InitializeDefaultProviders(); err != nil {
		logger.SysErrorf("failed to initialize default providers: " + err.Error())
	} else {
		logger.SysLog("providers initialized with default providers")
	}

	// Initialize i18n
	if err := i18n.Init(); err != nil {
		logger.FatalLog("failed to initialize i18n: " + err.Error())
	}

	// Initialize HTTP server
	server := gin.New()
	server.Use(gin.Recovery())
	// This will cause SSE not to work!!!
	//server.Use(gzip.Gzip(gzip.DefaultCompression))
	server.Use(middleware.RequestId())
	server.Use(middleware.Language())
	middleware.SetUpLogger(server)
	// Initialize session store
	store := cookie.NewStore([]byte(config.SessionSecret))
	server.Use(sessions.Sessions("session", store))

	router.SetRouter(server, buildFS)
	var port = os.Getenv("PORT")
	if port == "" {
		port = strconv.Itoa(*common.Port)
	}
	logger.SysLogf("server started on http://localhost:%s", port)
	err = server.Run(":" + port)
	if err != nil {
		logger.FatalLog("failed to start HTTP server: " + err.Error())
	}
}
