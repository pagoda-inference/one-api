package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common"
	"github.com/songquanpeng/one-api/common/config"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// TokenRateLimiter implements rate limiting per token
type TokenRateLimiter struct {
	// Concurrency tracking per token
	concurrentRequests map[int]*int32
}

// NewTokenRateLimiter creates a new token rate limiter
func NewTokenRateLimiter() *TokenRateLimiter {
	return &TokenRateLimiter{
		concurrentRequests: make(map[int]*int32),
	}
}

// Global token rate limiter instance
var globalTokenRateLimiter *TokenRateLimiter

// GetTokenRateLimiter returns the global token rate limiter
func GetTokenRateLimiter() *TokenRateLimiter {
	if globalTokenRateLimiter == nil {
		globalTokenRateLimiter = NewTokenRateLimiter()
	}
	return globalTokenRateLimiter
}

// checkRpm checks if the RPM limit is exceeded for a token
func checkRpm(tokenId int, limit int) (bool, int) {
	if limit <= 0 || !common.RedisEnabled {
		return true, 0
	}

	ctx := context.Background()
	// Key format: ratelimit:rpm:{token_id}:{minute}
	now := time.Now()
	minute := now.Unix() / 60
	key := fmt.Sprintf("ratelimit:rpm:%d:%d", tokenId, minute)

	count, err := common.RDB.Incr(ctx, key).Result()
	if err != nil {
		logger.SysError("Redis Incr error in RPM check: " + err.Error())
		return true, 0 // Allow on error
	}

	// Set expiry if this is the first request in this minute
	if count == 1 {
		common.RDB.Expire(ctx, key, time.Minute*2)
	}

	return count <= int64(limit), int(count)
}

// checkTpm checks if the TPM limit is exceeded for a token
func checkTpm(tokenId int, model string, estimatedTokens int, limit int) (bool, int) {
	if limit <= 0 || !common.RedisEnabled {
		return true, 0
	}

	ctx := context.Background()
	// Key format: ratelimit:tpm:{token_id}:{minute}
	now := time.Now()
	minute := now.Unix() / 60
	key := fmt.Sprintf("ratelimit:tpm:%d:%d", tokenId, minute)

	// Add estimated tokens to the counter
	count, err := common.RDB.IncrBy(ctx, key, int64(estimatedTokens)).Result()
	if err != nil {
		logger.SysError("Redis IncrBy error in TPM check: " + err.Error())
		return true, 0 // Allow on error
	}

	// Set expiry if this is the first write
	if count == int64(estimatedTokens) {
		common.RDB.Expire(ctx, key, time.Minute*2)
	}

	return count <= int64(limit), int(count)
}

// acquireConcurrency acquires a concurrency slot for a token
func (rl *TokenRateLimiter) acquireConcurrency(tokenId int, limit int) bool {
	if limit <= 0 {
		return true
	}

	// Get or create counter for this token
	rlPtr, exists := rl.concurrentRequests[tokenId]
	if !exists {
		var newCounter int32 = 0
		rl.concurrentRequests[tokenId] = &newCounter
		rlPtr = &newCounter
	}

	// Check current count
	current := atomic.LoadInt32(rlPtr)
	if current >= int32(limit) {
		return false
	}

	// Try to increment
	if !atomic.CompareAndSwapInt32(rlPtr, current, current+1) {
		// CAS failed, another request won the race
		return false
	}

	return true
}

// releaseConcurrency releases a concurrency slot for a token
func (rl *TokenRateLimiter) releaseConcurrency(tokenId int) {
	rlPtr, exists := rl.concurrentRequests[tokenId]
	if !exists {
		return
	}

	atomic.AddInt32(rlPtr, -1)
}

// TokenRateLimitMiddleware returns a Gin middleware for token-based rate limiting
func TokenRateLimitMiddleware() gin.HandlerFunc {
	rl := GetTokenRateLimiter()

	return func(c *gin.Context) {
		// Skip rate limiting if Redis is not enabled
		if !common.RedisEnabled {
			c.Next()
			return
		}

		tokenId := c.GetInt(ctxkey.TokenId)
		if tokenId == 0 {
			c.Next()
			return
		}

		// Get token from cache/database
		token, err := model.GetTokenById(tokenId)
		if err != nil {
			c.Next()
			return
		}

		// Check RPM (Requests Per Minute)
		if token.RateLimitRpm > 0 {
			allowed, current := checkRpm(tokenId, token.RateLimitRpm)
			if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Rate limit exceeded: %d requests per minute (current: %d)", token.RateLimitRpm, current),
						"type":    "rate_limit_error",
						"code":    "rpm_exceeded",
					},
				})
				c.Abort()
				return
			}
		}

		// Check TPM (Tokens Per Minute) - estimate based on request body
		if token.RateLimitTpm > 0 {
			estimatedTokens := estimateTokensFromRequest(c)
			allowed, current := checkTpm(tokenId, c.GetString(ctxkey.RequestModel), estimatedTokens, token.RateLimitTpm)
			if !allowed {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Token rate limit exceeded: %d tokens per minute (current: %d)", token.RateLimitTpm, current),
						"type":    "rate_limit_error",
						"code":    "tpm_exceeded",
					},
				})
				c.Abort()
				return
			}
		}

		// Check concurrent requests
		if token.RateLimitConcurrent > 0 {
			if !rl.acquireConcurrency(tokenId, token.RateLimitConcurrent) {
				c.JSON(http.StatusTooManyRequests, gin.H{
					"error": gin.H{
						"message": fmt.Sprintf("Concurrent request limit exceeded: %d concurrent requests", token.RateLimitConcurrent),
						"type":    "rate_limit_error",
						"code":    "concurrent_exceeded",
					},
				})
				c.Abort()
				return
			}
			defer rl.releaseConcurrency(tokenId)
		}

		c.Next()
	}
}

// estimateTokensFromRequest estimates the number of tokens in a request
func estimateTokensFromRequest(c *gin.Context) int {
	// Get request body
	body, err := c.GetRawData()
	if err != nil || len(body) == 0 {
		return 100 // Default estimate
	}

	// Restore body for later use
	c.Request.Body = &readCloser{body: body}

	// Try to parse as JSON to get messages
	var request struct {
		Messages []struct {
			Content string `json:"content"`
		} `json:"messages"`
		MaxTokens int `json:"max_tokens"`
	}

	if err := json.Unmarshal(body, &request); err == nil {
		totalTokens := 0

		// Estimate tokens from messages (roughly 4 chars per token)
		for _, msg := range request.Messages {
			totalTokens += len(msg.Content) / 4
		}

		// Add expected output tokens if specified
		if request.MaxTokens > 0 {
			totalTokens += request.MaxTokens
		} else {
			// Default output estimate
			totalTokens += config.PreConsumedQuota
		}

		return totalTokens
	}

	// Fallback: estimate based on body length
	return len(body) / 4
}

// readCloser wraps a byte slice to implement io.ReadCloser
type readCloser struct {
	body   []byte
	offset int
}

func (rc *readCloser) Read(p []byte) (n int, err error) {
	if rc.offset >= len(rc.body) {
		return 0, nil
	}
	n = copy(p, rc.body[rc.offset:])
	rc.offset += n
	return n, nil
}

func (rc *readCloser) Close() error {
	return nil
}