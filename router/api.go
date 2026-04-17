package router

import (
	"github.com/pagoda-inference/one-api/controller"
	"github.com/pagoda-inference/one-api/controller/auth"
	"github.com/pagoda-inference/one-api/middleware"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
)

func SetApiRouter(router *gin.Engine) {
	apiRouter := router.Group("/api")
	apiRouter.Use(gzip.Gzip(gzip.DefaultCompression))
	apiRouter.Use(middleware.GlobalAPIRateLimit())
	{
		apiRouter.GET("/status", controller.GetStatus)
		apiRouter.GET("/models", middleware.UserAuth(), controller.DashboardListModels)
		apiRouter.GET("/notice", controller.GetNotice)
		apiRouter.GET("/about", controller.GetAbout)
		apiRouter.GET("/home_page_content", controller.GetHomePageContent)
		apiRouter.GET("/api-docs", controller.GetApiDocs)
		apiRouter.GET("/verification", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendEmailVerification)
		apiRouter.GET("/reset_password", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.SendPasswordResetEmail)
		apiRouter.POST("/user/reset", middleware.CriticalRateLimit(), controller.ResetPassword)
		apiRouter.GET("/oauth/github", middleware.CriticalRateLimit(), auth.GitHubOAuth)
		apiRouter.GET("/oauth/oidc", middleware.CriticalRateLimit(), auth.OidcAuth)
		apiRouter.GET("/oauth/lark", middleware.CriticalRateLimit(), auth.LarkOAuth)
		apiRouter.GET("/oauth/state", middleware.CriticalRateLimit(), auth.GenerateOAuthCode)
		apiRouter.GET("/oauth/wechat", middleware.CriticalRateLimit(), auth.WeChatAuth)

		// Public route for Lark OAuth apps (used by login page)
		apiRouter.GET("/lark-apps", controller.GetEnabledLarkOAuthApps)
		apiRouter.GET("/oauth/wechat/bind", middleware.CriticalRateLimit(), middleware.UserAuth(), auth.WeChatBind)
		apiRouter.GET("/oauth/email/bind", middleware.CriticalRateLimit(), middleware.UserAuth(), controller.EmailBind)
		apiRouter.POST("/topup", middleware.AdminAuth(), controller.AdminTopUp)

		// Payment callback routes (no auth - called by payment providers)
		apiRouter.POST("/callback/alipay", controller.AlipayNotify)
		apiRouter.POST("/callback/wechat", controller.WechatPayNotify)

		// Dashboard v2 (enhanced)
		apiRouter.GET("/dashboard", middleware.UserAuth(), controller.GetUserDashboardV2)
		apiRouter.GET("/usage/detail", middleware.UserAuth(), controller.GetUsageDetail)

		userRoute := apiRouter.Group("/user")
		{
			userRoute.POST("/register", middleware.CriticalRateLimit(), middleware.TurnstileCheck(), controller.Register)
			userRoute.POST("/login", middleware.CriticalRateLimit(), controller.Login)
			userRoute.GET("/logout", controller.Logout)

			selfRoute := userRoute.Group("/")
			selfRoute.Use(middleware.UserAuth())
			{
				selfRoute.GET("/dashboard", controller.GetUserDashboard)
				selfRoute.GET("/self", controller.GetSelf)
				selfRoute.PUT("/self", controller.UpdateSelf)
				selfRoute.DELETE("/self", controller.DeleteSelf)
				selfRoute.GET("/token", controller.GenerateAccessToken)
				selfRoute.GET("/aff", controller.GetAffCode)
				selfRoute.POST("/topup", controller.TopUp)
				selfRoute.GET("/available_models", controller.GetUserAvailableModels)
				selfRoute.GET("/signin/records", controller.GetSignInRecords)
				selfRoute.POST("/signin", controller.SignIn)
				selfRoute.GET("/notifications", controller.GetUserNotifications)
				selfRoute.GET("/notifications/unread-count", controller.GetUnreadNotificationCount)
				selfRoute.PUT("/notifications/:id/read", controller.MarkNotificationAsRead)
				selfRoute.PUT("/notifications/read-all", controller.MarkAllNotificationsAsRead)
			}

			// Payment routes
			paymentRoute := userRoute.Group("/")
			paymentRoute.Use(middleware.UserAuth())
			{
				paymentRoute.POST("/topup/create", controller.CreateTopupOrder)
				paymentRoute.GET("/topup", controller.GetTopupOrders)
				paymentRoute.GET("/topup/:id", controller.GetTopupOrder)
				paymentRoute.POST("/topup/:id/cancel", controller.CancelTopupOrder)
				paymentRoute.POST("/invoice", controller.CreateInvoice)
				paymentRoute.GET("/invoice", controller.GetInvoices)
				paymentRoute.GET("/invoice/:id", controller.GetInvoice)
			}

			// Market routes
			marketRoute := userRoute.Group("/market")
			marketRoute.Use(middleware.UserAuth())
			{
				marketRoute.GET("/models", controller.GetMarketModels)
				marketRoute.GET("/models/:id", controller.GetMarketModel)
				marketRoute.GET("/models/:id/pricing", controller.GetModelPricing)
				marketRoute.GET("/models/:id/trial", controller.GetModelTrial)
				marketRoute.POST("/models/:id/trial", controller.StartModelTrial)
				marketRoute.GET("/providers", controller.GetMarketProviders)
				marketRoute.GET("/groups", controller.GetMarketGroups)
				marketRoute.GET("/groups/:id/models", controller.GetMarketModelsByGroup)
				marketRoute.GET("/stats", controller.GetMarketStats)
				marketRoute.GET("/calculate", controller.CalculatePrice)
				marketRoute.GET("/trials", controller.GetUserTrials)
			}
		}

		// Admin market routes
		adminMarketRoute := apiRouter.Group("/admin/market")
		adminMarketRoute.Use(middleware.AdminAuth())
		{
			adminMarketRoute.POST("/pricing", controller.SetModelPricing)
			// ModelGroup CRUD is deprecated - use channels.group for provider concept
		}

		// Tenant routes (multi-tenancy)
		tenantRoute := apiRouter.Group("/tenant")
		tenantRoute.Use(middleware.UserAuth())
		{
			tenantRoute.POST("/", controller.CreateTenant)
			tenantRoute.GET("/", controller.GetMyTenants)
			tenantRoute.GET("/:id", controller.GetTenant)
			tenantRoute.PUT("/:id", controller.UpdateTenant)
			tenantRoute.GET("/:id/users", controller.GetTenantUsersAPI)
			tenantRoute.POST("/:id/users", controller.InviteUser)
			tenantRoute.DELETE("/:id/users/:userId", controller.RemoveUser)
			tenantRoute.PUT("/:id/users/:userId", controller.UpdateUserRole)
			tenantRoute.POST("/:id/quota", controller.AllocateUserQuotaAPI)
			tenantRoute.GET("/:id/audit", controller.GetAuditLogsAPI)
			tenantRoute.POST("/:id/leave", controller.LeaveTenant)
		}

		// Company routes (for Platform/Company/Department/Team hierarchy)
		companyRoute := apiRouter.Group("/company")
		companyRoute.Use(middleware.UserAuth())
		{
			companyRoute.POST("/", controller.CreateCompany)
			companyRoute.GET("/", controller.GetAllCompanies)
			companyRoute.GET("/:id", controller.GetCompany)
			companyRoute.PUT("/:id", controller.UpdateCompany)
			companyRoute.DELETE("/:id", controller.DeleteCompany)
			// Department routes under company
			companyRoute.POST("/:id/departments", controller.CreateDepartment)
			companyRoute.GET("/:id/departments", controller.GetDepartments)
		}

		// Department routes (standalone)
		departmentRoute := apiRouter.Group("/department")
		departmentRoute.Use(middleware.UserAuth())
		{
			departmentRoute.GET("/:id", controller.GetDepartment)
			departmentRoute.PUT("/:id", controller.UpdateDepartment)
			departmentRoute.DELETE("/:id", controller.DeleteDepartment)
		}

		// Admin routes
		adminRoute := apiRouter.Group("/admin")
		adminRoute.Use(middleware.AdminAuth())
		{
			adminRoute.GET("/", controller.GetAllUsers)
			adminRoute.GET("/search", controller.SearchUsers)
			adminRoute.GET("/:id", controller.GetUser)
			adminRoute.POST("/", controller.CreateUser)
			adminRoute.POST("/manage", controller.ManageUser)
			adminRoute.PUT("/:id", controller.UpdateUser)
			adminRoute.DELETE("/:id", controller.DeleteUser)

			// Ops routes
			adminRoute.GET("/ops/stats", controller.GetOpsStats)
			adminRoute.GET("/ops/revenue", controller.GetOpsRevenue)
			adminRoute.GET("/ops/usage", controller.GetOpsUsage)
			adminRoute.GET("/ops/users", controller.GetOpsUsers)
			adminRoute.GET("/channels/health", controller.GetChannelHealth)
			adminRoute.GET("/alerts/config", controller.GetAlertConfig)
			adminRoute.PUT("/alerts/config", controller.UpdateAlertConfig)
			adminRoute.GET("/notifications", controller.GetAllNotifications)
			adminRoute.POST("/notifications", controller.CreateNotification)
			adminRoute.DELETE("/notifications/:id", controller.DeleteNotification)
			adminRoute.GET("/system/health", controller.GetSystemHealth)
			adminRoute.GET("/reports/export", controller.ExportReport)
			adminRoute.GET("/tenants", controller.GetAllTenantsForAdmin)

			// Model management routes
			adminRoute.GET("/models", controller.AdminListModels)
			adminRoute.GET("/models/types", controller.GetModelTypes)
			adminRoute.GET("/models/statuses", controller.GetModelStatuses)
			adminRoute.GET("/models/model", controller.GetModel)
			adminRoute.POST("/models", controller.CreateModel)
			adminRoute.PUT("/models/model", controller.UpdateModel)
			adminRoute.DELETE("/models/model", controller.DeleteModel)
			adminRoute.POST("/models/batch-delete", controller.BatchDeleteModels)
			adminRoute.POST("/models/upload-logo", controller.UploadModelLogo)

			// Provider management routes
			adminRoute.GET("/providers", controller.GetProviders)
			adminRoute.GET("/providers/statuses", controller.GetProviderStatuses)
			adminRoute.GET("/providers/:id", controller.GetProvider)
			adminRoute.POST("/providers", controller.CreateProvider)
			adminRoute.PUT("/providers/:id", controller.UpdateProvider)
			adminRoute.DELETE("/providers/:id", controller.DeleteProvider)
			adminRoute.POST("/providers/upload-logo", controller.UploadProviderLogo)

			// Logo management routes
			adminRoute.GET("/logos", controller.ListLogos)
			adminRoute.DELETE("/logos", controller.DeleteLogo)

			// Lark OAuth app management routes
			adminRoute.GET("/lark-apps", controller.GetLarkOAuthApps)
			adminRoute.POST("/lark-apps", controller.CreateLarkOAuthApp)
			adminRoute.PUT("/lark-apps/:id", controller.UpdateLarkOAuthApp)
			adminRoute.DELETE("/lark-apps/:id", controller.DeleteLarkOAuthApp)
		}

		optionRoute := apiRouter.Group("/option")
		optionRoute.Use(middleware.RootAuth())
		{
			optionRoute.GET("/", controller.GetOptions)
			optionRoute.PUT("/", controller.UpdateOption)
		}
		channelRoute := apiRouter.Group("/channel")
		channelRoute.Use(middleware.AdminAuth())
		{
			channelRoute.GET("/", controller.GetAllChannels)
			channelRoute.GET("/groups", controller.GetDistinctChannelGroups)
			channelRoute.GET("/search", controller.SearchChannels)
			channelRoute.GET("/models", controller.ListAllModels)
			channelRoute.GET("/:id", controller.GetChannel)
			channelRoute.GET("/test", controller.TestChannels)
			channelRoute.GET("/test/:id", controller.TestChannel)
			channelRoute.GET("/update_balance", controller.UpdateAllChannelsBalance)
			channelRoute.GET("/update_balance/:id", controller.UpdateChannelBalance)
			channelRoute.POST("/", controller.AddChannel)
			channelRoute.PUT("/", controller.UpdateChannel)
			channelRoute.DELETE("/disabled", controller.DeleteDisabledChannel)
			channelRoute.DELETE("/:id", controller.DeleteChannel)
		}
		tokenRoute := apiRouter.Group("/token")
		tokenRoute.Use(middleware.UserAuth())
		{
			tokenRoute.GET("/", controller.GetAllTokens)
			tokenRoute.GET("/search", controller.SearchTokens)
			tokenRoute.GET("/:id", controller.GetToken)
			tokenRoute.POST("/", controller.AddToken)
			tokenRoute.PUT("/:id", controller.UpdateToken)
			tokenRoute.DELETE("/:id", controller.DeleteToken)
		}
		redemptionRoute := apiRouter.Group("/redemption")
		redemptionRoute.Use(middleware.AdminAuth())
		{
			redemptionRoute.GET("/", controller.GetAllRedemptions)
			redemptionRoute.GET("/search", controller.SearchRedemptions)
			redemptionRoute.GET("/:id", controller.GetRedemption)
			redemptionRoute.POST("/", controller.AddRedemption)
			redemptionRoute.PUT("/", controller.UpdateRedemption)
			redemptionRoute.DELETE("/:id", controller.DeleteRedemption)
		}
		logRoute := apiRouter.Group("/log")
		logRoute.GET("/", middleware.AdminAuth(), controller.GetAllLogs)
		logRoute.DELETE("/", middleware.AdminAuth(), controller.DeleteHistoryLogs)
		logRoute.GET("/stat", middleware.AdminAuth(), controller.GetLogsStat)
		logRoute.GET("/self/stat", middleware.UserAuth(), controller.GetLogsSelfStat)
		logRoute.GET("/search", middleware.AdminAuth(), controller.SearchAllLogs)
		logRoute.GET("/self", middleware.UserAuth(), controller.GetUserLogs)
		logRoute.GET("/self/search", middleware.UserAuth(), controller.SearchUserLogs)
		usageRoute := apiRouter.Group("/usage")
		usageRoute.Use(middleware.UserAuth())
		{
			usageRoute.GET("/summary", controller.GetUsageSummary)
			usageRoute.GET("/by-token", controller.GetUsageByToken)
			usageRoute.GET("/by-model", controller.GetUsageByModel)
			usageRoute.GET("/by-channel", controller.GetUsageByChannel)
			usageRoute.GET("/by-hour", controller.GetUsageByHour)
			usageRoute.GET("/daily", controller.GetUsageDaily)
		}
		adminUsageRoute := apiRouter.Group("/admin/usage")
		adminUsageRoute.Use(middleware.AdminTokenAuth())
		{
			adminUsageRoute.GET("/summary", controller.AdminGetUsageSummary)
			adminUsageRoute.GET("/by-users", controller.AdminGetUsageByUsers)
			adminUsageRoute.GET("/by-models", controller.AdminGetUsageByModels)
		}
		groupRoute := apiRouter.Group("/group")
		groupRoute.Use(middleware.AdminAuth())
		{
			groupRoute.GET("/", controller.GetGroups)
		}
	}
}