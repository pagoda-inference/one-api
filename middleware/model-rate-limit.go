package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
)

// ModelRateLimiter implements rate limiting per model
type ModelRateLimiter struct{}

// Global model rate limiter instance
var globalModelRateLimiter *ModelRateLimiter

// GetModelRateLimiter returns the global model rate limiter
func GetModelRateLimiter() *ModelRateLimiter {
	if globalModelRateLimiter == nil {
		globalModelRateLimiter = &ModelRateLimiter{}
	}
	return globalModelRateLimiter
}

// checkModelRpm checks if the RPM limit is exceeded for a model
// All users share the same counter for a given model
func checkModelRpm(modelName string, limit int) (bool, int) {
	if limit <= 0 || !common.RedisEnabled {
		return true, 0
	}

	ctx := context.Background()
	// Key format: ratelimit:model:rpm:{model_name}:{minute}
	now := time.Now()
	minute := now.Unix() / 60
	key := fmt.Sprintf("ratelimit:model:rpm:%s:%d", modelName, minute)

	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		logger.SysError("Redis Incr error in model RPM check: " + err.Error())
		return true, 0 // Allow on error
	}

	// Set expiry if this is the first request in this minute
	if count == 1 {
		common.RDB.Expire(ctx, key, time.Minute*2)
	}

	return count <= int64(limit), int(count)
}

// checkModelTpm checks if the TPM limit is exceeded for a model
// All users share the same counter for a given model
func checkModelTpm(modelName string, estimatedTokens int, limit int) (bool, int) {
	if limit <= 0 || !common.RedisEnabled {
		return true, 0
	}

	ctx := context.Background()
	// Key format: ratelimit:model:tpm:{model_name}:{minute}
	now := time.Now()
	minute := now.Unix() / 60
	key := fmt.Sprintf("ratelimit:model:tpm:%s:%d", modelName, minute)

	// Add estimated tokens to the counter
	count, err := common.RDB.IncrBy(ctx, key, int64(estimatedTokens)).Result()
	if err != nil {
		logger.SysError("Redis IncrBy error in model TPM check: " + err.Error())
		return true, 0 // Allow on error
	}

	// Set expiry if this is the first write
	if count == int64(estimatedTokens) {
		common.RDB.Expire(ctx, key, time.Minute*2)
	}

	return count <= int64(limit), int(count)
}

// ModelRateLimitMiddleware returns a Gin middleware for model-based rate limiting
// This limits requests per model across ALL users (per-model aggregation)
func ModelRateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !common.RedisEnabled {
			c.Next()
			return
		}

		modelName := c.GetString(ctxkey.RequestModel)
		if modelName == "" {
			c.Next()
			return
		}

		// Get model info to check rate limits
		modelInfo, err := model.GetModelById(modelName)
		if err != nil || modelInfo == nil {
			// Model not found in model_info, skip model rate limiting
			c.Next()
			return
		}

		// Check RPM (Requests Per Minute) - per model, all users share
		if modelInfo.RateLimitRPM > 0 {
			allowed, current := checkModelRpm(modelName, modelInfo.RateLimitRPM)
			if !allowed {
				logger.Debugf(c.Request.Context(), "Model rate limit exceeded: model=%s, rpm=%d, current=%d", modelName, modelInfo.RateLimitRPM, current)
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Model rate limit exceeded: %d requests per minute (current: %d)", modelInfo.RateLimitRPM, current),
						"type":    "rate_limit_error",
						"code":    "model_rpm_exceeded",
					},
				})
				c.Abort()
				return
			}
		}

		// Check TPM (Tokens Per Minute) - per model, all users share
		if modelInfo.RateLimitTPM > 0 {
			estimatedTokens := estimateTokensFromRequest(c)
			allowed, current := checkModelTpm(modelName, estimatedTokens, modelInfo.RateLimitTPM)
			if !allowed {
				logger.Debugf(c.Request.Context(), "Model token rate limit exceeded: model=%s, tpm=%d, current=%d", modelName, modelInfo.RateLimitTPM, current)
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Model token rate limit exceeded: %d tokens per minute (current: %d)", modelInfo.RateLimitTPM, current),
						"type":    "rate_limit_error",
						"code":    "model_tpm_exceeded",
					},
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
