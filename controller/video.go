package controller

import (
	"github.com/gin-gonic/gin"

	videocontroller "github.com/pagoda-inference/one-api/relay/controller"
)

// This file exposes the video HTTP entry points at the top-level controller
// package so the router can reference them as controller.* (consistent with
// Relay / RelayResponses / RelayAnthropicPassthrough defined in relay.go). Each
// entry is a thin pass-through to the implementation in relay/controller,
// which lives beside RelayTextHelper / RelayImageHelper / RelayAudioHelper.

// RelayVideoRetrieve handles GET /v1/videos/:id.
func RelayVideoRetrieve(c *gin.Context) { videocontroller.RelayVideoRetrieve(c) }

// RelayVideoList handles GET /v1/videos.
func RelayVideoList(c *gin.Context) { videocontroller.RelayVideoList(c) }

// RelayVideoDelete handles DELETE /v1/videos/:id.
func RelayVideoDelete(c *gin.Context) { videocontroller.RelayVideoDelete(c) }

// RelayVideoContent handles GET /v1/videos/:id/content (same-origin video byte proxy).
func RelayVideoContent(c *gin.Context) { videocontroller.RelayVideoContent(c) }

// Legacy Ark-style handlers (/api/v1/contents/generations/tasks*) kept for
// backward compatibility; they delegate to the shared relay/controller impl.

// CreateVideoGenerationTask handles POST /api/v1/contents/generations/tasks.
func CreateVideoGenerationTask(c *gin.Context) { videocontroller.CreateVideoGenerationTask(c) }

// GetVideoGenerationTask handles GET /api/v1/contents/generations/tasks/:id.
func GetVideoGenerationTask(c *gin.Context) { videocontroller.GetVideoGenerationTask(c) }

// ListVideoGenerationTasks handles GET /api/v1/contents/generations/tasks.
func ListVideoGenerationTasks(c *gin.Context) { videocontroller.ListVideoGenerationTasks(c) }

// DeleteVideoGenerationTask handles DELETE /api/v1/contents/generations/tasks/:id.
func DeleteVideoGenerationTask(c *gin.Context) { videocontroller.DeleteVideoGenerationTask(c) }

// ProxyVideoGenerationTaskContent proxies generated video bytes through a
// same-origin endpoint for the legacy Ark and market routes.
func ProxyVideoGenerationTaskContent(c *gin.Context) {
	videocontroller.ProxyVideoGenerationTaskContent(c)
}
