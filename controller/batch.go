package controller

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/songquanpeng/one-api/common/ctxkey"
	"github.com/songquanpeng/one-api/common/helper"
	"github.com/songquanpeng/one-api/common/logger"
	"github.com/songquanpeng/one-api/model"
)

// Batch config constants
const (
	MaxFileSize         = 100 * 1024 * 1024 // 100MB
	DefaultBatchExpiry  = 24 * time.Hour
	MaxBatchExpiry      = 7 * 24 * time.Hour
	BatchStoragePathEnv = "BATCH_STORAGE_PATH"
)

// getBatchStoragePath returns the storage path for batch files
func getBatchStoragePath() string {
	path := os.Getenv(BatchStoragePathEnv)
	if path == "" {
		path = "./batch_files"
	}
	return path
}

// ensureStorageDir ensures the storage directory exists
func ensureStorageDir() error {
	path := getBatchStoragePath()
	return os.MkdirAll(path, 0755)
}

// UploadFile handles POST /v1/files
func UploadFile(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	tokenId := c.GetInt(ctxkey.TokenId)

	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	// Get purpose from form data
	purpose := c.PostForm("purpose")
	if purpose == "" {
		purpose = model.FilePurposeBatch
	}

	// Validate purpose
	if purpose != model.FilePurposeBatch && purpose != model.FilePurposeBatchOutput {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid purpose. Must be 'batch' or 'batch_output'",
				"type":    "invalid_request_error",
				"code":    "invalid_purpose",
			},
		})
		return
	}

	// Get file from form
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "No file uploaded",
				"type":    "invalid_request_error",
				"code":    "no_file",
			},
		})
		return
	}
	defer file.Close()

	// Check file size
	if header.Size > MaxFileSize {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("File size exceeds maximum allowed: %d bytes", MaxFileSize),
				"type":    "invalid_request_error",
				"code":    "file_too_large",
			},
		})
		return
	}

	// Ensure storage directory exists
	if err := ensureStorageDir(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create storage directory",
				"type":    "server_error",
				"code":    "storage_error",
			},
		})
		return
	}

	// Create file record
	batchFile := &model.BatchFile{
		UserId:      userId,
		TokenId:     tokenId,
		Filename:    header.Filename,
		Purpose:     purpose,
		Bytes:       header.Size,
		Status:      model.FileStatusUploaded,
		CreatedAt:   helper.GetTimestamp(),
		ExpiresAt:   helper.GetTimestamp() + int64(DefaultBatchExpiry.Seconds()),
	}

	if err := batchFile.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create file record",
				"type":    "server_error",
				"code":    "database_error",
			},
		})
		return
	}

	// Generate storage path
	storagePath := filepath.Join(getBatchStoragePath(), fmt.Sprintf("%d_%d_%s", userId, batchFile.Id, header.Filename))
	batchFile.StoragePath = storagePath

	// Save file to disk
	dst, err := os.Create(storagePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to save file",
				"type":    "server_error",
				"code":    "storage_error",
			},
		})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to save file content",
				"type":    "server_error",
				"code":    "storage_error",
			},
		})
		return
	}

	// Update file record with storage path
	if err := batchFile.Update(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to update file record",
				"type":    "server_error",
				"code":    "database_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":        fmt.Sprintf("file-%d", batchFile.Id),
		"object":    "file",
		"bytes":     batchFile.Bytes,
		"created_at": batchFile.CreatedAt,
		"filename":  batchFile.Filename,
		"purpose":   batchFile.Purpose,
		"status":    batchFile.Status,
	})
}

// ListFiles handles GET /v1/files
func ListFiles(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	purpose := c.Query("purpose")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	after, _ := strconv.Atoi(c.Query("after"))

	files, err := model.GetUserFiles(userId, purpose, limit, after)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to list files",
				"type":    "server_error",
				"code":    "database_error",
			},
		})
		return
	}

	data := make([]gin.H, len(files))
	for i, f := range files {
		data[i] = gin.H{
			"id":         fmt.Sprintf("file-%d", f.Id),
			"object":     "file",
			"bytes":      f.Bytes,
			"created_at": f.CreatedAt,
			"filename":   f.Filename,
			"purpose":    f.Purpose,
			"status":     f.Status,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

// RetrieveFile handles GET /v1/files/:id
func RetrieveFile(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	fileIdStr := c.Param("id")
	fileIdStr = strings.TrimPrefix(fileIdStr, "file-")
	fileId, err := strconv.Atoi(fileIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid file ID",
				"type":    "invalid_request_error",
				"code":    "invalid_id",
			},
		})
		return
	}

	file, err := model.GetUserFileById(fileId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "File not found",
				"type":    "invalid_request_error",
				"code":    "file_not_found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":         fmt.Sprintf("file-%d", file.Id),
		"object":     "file",
		"bytes":      file.Bytes,
		"created_at": file.CreatedAt,
		"filename":   file.Filename,
		"purpose":    file.Purpose,
		"status":     file.Status,
	})
}

// RetrieveFileContent handles GET /v1/files/:id/content
func RetrieveFileContent(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	fileIdStr := c.Param("id")
	fileIdStr = strings.TrimPrefix(fileIdStr, "file-")
	fileId, err := strconv.Atoi(fileIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid file ID",
				"type":    "invalid_request_error",
				"code":    "invalid_id",
			},
		})
		return
	}

	file, err := model.GetUserFileById(fileId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "File not found",
				"type":    "invalid_request_error",
				"code":    "file_not_found",
			},
		})
		return
	}

	// Check if file exists on disk
	if _, err := os.Stat(file.StoragePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "File content not found",
				"type":    "server_error",
				"code":    "file_missing",
			},
		})
		return
	}

	// Set content disposition header
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=%s", file.Filename))
	c.File(file.StoragePath)
}

// DeleteFile handles DELETE /v1/files/:id
func DeleteFile(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	fileIdStr := c.Param("id")
	fileIdStr = strings.TrimPrefix(fileIdStr, "file-")
	fileId, err := strconv.Atoi(fileIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid file ID",
				"type":    "invalid_request_error",
				"code":    "invalid_id",
			},
		})
		return
	}

	file, err := model.GetUserFileById(fileId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "File not found",
				"type":    "invalid_request_error",
				"code":    "file_not_found",
			},
		})
		return
	}

	// Delete file from disk
	if file.StoragePath != "" {
		os.Remove(file.StoragePath)
	}

	// Delete file record
	if err := file.Delete(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to delete file",
				"type":    "server_error",
				"code":    "database_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":      fmt.Sprintf("file-%d", file.Id),
		"object":  "file",
		"deleted": true,
	})
}

// CreateBatchRequest represents the request body for creating a batch
type CreateBatchRequest struct {
	InputFileId       string `json:"input_file_id"`
	Endpoint          string `json:"endpoint"`
	CompletionWindow string `json:"completion_window"`
	Metadata          string `json:"metadata"`
}

// CreateBatch handles POST /v1/batches
func CreateBatch(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	tokenId := c.GetInt(ctxkey.TokenId)

	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	var req CreateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid request body",
				"type":    "invalid_request_error",
				"code":    "invalid_body",
			},
		})
		return
	}

	// Validate input file ID
	fileIdStr := strings.TrimPrefix(req.InputFileId, "file-")
	fileId, err := strconv.Atoi(fileIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid input_file_id",
				"type":    "invalid_request_error",
				"code":    "invalid_file_id",
			},
		})
		return
	}

	// Verify file ownership and purpose
	file, err := model.GetUserFileById(fileId, userId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Input file not found",
				"type":    "invalid_request_error",
				"code":    "file_not_found",
			},
		})
		return
	}

	if file.Purpose != model.FilePurposeBatch {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "File must have purpose 'batch'",
				"type":    "invalid_request_error",
				"code":    "invalid_purpose",
			},
		})
		return
	}

	// Validate endpoint
	validEndpoints := map[string]bool{
		"/v1/chat/completions": true,
		"/v1/completions":      true,
		"/v1/embeddings":       true,
	}
	if !validEndpoints[req.Endpoint] {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid endpoint. Must be /v1/chat/completions, /v1/completions, or /v1/embeddings",
				"type":    "invalid_request_error",
				"code":    "invalid_endpoint",
			},
		})
		return
	}

	// Validate completion window
	if req.CompletionWindow != "24h" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": "Invalid completion_window. Only '24h' is supported",
				"type":    "invalid_request_error",
				"code":    "invalid_completion_window",
			},
		})
		return
	}

	// Count requests in the input file
	requestCounts, err := countRequestsInFile(file.StoragePath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("Failed to parse input file: %v", err),
				"type":    "invalid_request_error",
				"code":    "invalid_file_format",
			},
		})
		return
	}

	// Create batch record
	batch := &model.Batch{
		UserId:            userId,
		TokenId:           tokenId,
		Status:            model.BatchStatusValidating,
		InputFileId:       fileId,
		Endpoint:          req.Endpoint,
		CompletionWindow:  req.CompletionWindow,
		CreatedAt:         helper.GetTimestamp(),
		ExpiresAt:         helper.GetTimestamp() + int64(DefaultBatchExpiry.Seconds()),
		RequestCounts:     &requestCounts,
		Metadata:          req.Metadata,
	}

	if err := batch.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to create batch",
				"type":    "server_error",
				"code":    "database_error",
			},
		})
		return
	}

	// Start batch processing asynchronously
	go processBatch(batch.Id)

	c.JSON(http.StatusOK, formatBatchResponse(batch))
}

// ListBatches handles GET /v1/batches
func ListBatches(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	status := c.Query("status")
	limit, _ := strconv.Atoi(c.Query("limit"))
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	after := c.Query("after")

	batches, err := model.GetUserBatches(userId, status, limit, after)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to list batches",
				"type":    "server_error",
				"code":    "database_error",
			},
		})
		return
	}

	data := make([]gin.H, len(batches))
	for i, b := range batches {
		data[i] = formatBatchResponse(b)
	}

	c.JSON(http.StatusOK, gin.H{
		"object": "list",
		"data":   data,
	})
}

// RetrieveBatch handles GET /v1/batches/:id
func RetrieveBatch(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	batchId := c.Param("id")

	batch, err := model.GetUserBatchById(batchId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "Batch not found",
				"type":    "invalid_request_error",
				"code":    "batch_not_found",
			},
		})
		return
	}

	c.JSON(http.StatusOK, formatBatchResponse(batch))
}

// CancelBatch handles POST /v1/batches/:id/cancel
func CancelBatch(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{
				"message": "Unauthorized",
				"type":    "invalid_request_error",
				"code":    "unauthorized",
			},
		})
		return
	}

	batchId := c.Param("id")

	batch, err := model.GetUserBatchById(batchId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{
				"message": "Batch not found",
				"type":    "invalid_request_error",
				"code":    "batch_not_found",
			},
		})
		return
	}

	// Check if batch can be cancelled
	if batch.Status == model.BatchStatusCompleted ||
		batch.Status == model.BatchStatusCancelled ||
		batch.Status == model.BatchStatusExpired {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{
				"message": fmt.Sprintf("Cannot cancel batch with status '%s'", batch.Status),
				"type":    "invalid_request_error",
				"code":    "cannot_cancel",
			},
		})
		return
	}

	if err := batch.Cancel(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{
				"message": "Failed to cancel batch",
				"type":    "server_error",
				"code":    "database_error",
			},
		})
		return
	}

	c.JSON(http.StatusOK, formatBatchResponse(batch))
}

// formatBatchResponse formats a batch for API response
func formatBatchResponse(b *model.Batch) gin.H {
	response := gin.H{
		"id":                b.Id,
		"object":            "batch",
		"endpoint":          b.Endpoint,
		"errors":            nil,
		"input_file_id":     fmt.Sprintf("file-%d", b.InputFileId),
		"completion_window": b.CompletionWindow,
		"status":            b.Status,
		"output_file_id":    nil,
		"error_file_id":     nil,
		"created_at":        b.CreatedAt,
		"in_progress_at":    b.InProgressAt,
		"expires_at":        b.ExpiresAt,
		"finalizing_at":     b.FinalizingAt,
		"completed_at":      b.CompletedAt,
		"cancelled_at":      b.CancelledAt,
		"failed_at":         b.FailedAt,
		"metadata":          b.Metadata,
	}

	if b.OutputFileId != nil {
		response["output_file_id"] = fmt.Sprintf("file-%d", *b.OutputFileId)
	}
	if b.ErrorFileId != nil {
		response["error_file_id"] = fmt.Sprintf("file-%d", *b.ErrorFileId)
	}
	if b.RequestCounts != nil {
		response["request_counts"] = gin.H{
			"total":     *b.RequestCounts,
			"completed": b.CompletedCounts,
			"failed":    b.FailedCounts,
		}
	}

	return response
}

// countRequestsInFile counts the number of requests in a JSONL file
func countRequestsInFile(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	count := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			count++
		}
	}
	return count, scanner.Err()
}

// processBatch processes a batch asynchronously
func processBatch(batchId string) {
	ctx := context.Background()

	// Get batch from database
	batch, err := model.GetBatchById(batchId)
	if err != nil {
		logger.SysErrorf("Failed to get batch %s: %v", batchId, err)
		return
	}

	// Update status to in_progress
	if err := batch.SetStatus(model.BatchStatusInProgress); err != nil {
		logger.SysErrorf("Failed to update batch status: %v", err)
		return
	}

	// Get input file
	inputFile, err := model.GetFileById(batch.InputFileId)
	if err != nil {
		logger.SysErrorf("Failed to get input file: %v", err)
		batch.SetStatus(model.BatchStatusFailed)
		return
	}

	// Open input file
	file, err := os.Open(inputFile.StoragePath)
	if err != nil {
		logger.SysErrorf("Failed to open input file: %v", err)
		batch.SetStatus(model.BatchStatusFailed)
		return
	}
	defer file.Close()

	// Create output file
	outputPath := filepath.Join(getBatchStoragePath(), fmt.Sprintf("output_%d_%s.jsonl", batch.UserId, batch.Id))
	outputFile, err := os.Create(outputPath)
	if err != nil {
		logger.SysErrorf("Failed to create output file: %v", err)
		batch.SetStatus(model.BatchStatusFailed)
		return
	}
	defer outputFile.Close()

	// Process each request
	scanner := bufio.NewScanner(file)
	completedCount := 0
	failedCount := 0
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Check if batch was cancelled
		updatedBatch, _ := model.GetBatchById(batchId)
		if updatedBatch.Status == model.BatchStatusCancelled {
			logger.SysLogf("Batch %s was cancelled", batchId)
			return
		}

		// Process the request
		result, err := processBatchLine(ctx, line, lineNum, batch)
		if err != nil {
			failedCount++
			// Write error result
			errorResult := gin.H{
				"id":            fmt.Sprintf("request-%d", lineNum),
				"custom_id":     fmt.Sprintf("request-%d", lineNum),
				"error":         gin.H{"message": err.Error()},
				"response":      nil,
				"status_code":   500,
			}
			resultBytes, _ := json.Marshal(errorResult)
			outputFile.WriteString(string(resultBytes) + "\n")
		} else {
			completedCount++
			outputFile.WriteString(result + "\n")
		}

		// Update counts
		batch.CompletedCounts = &completedCount
		batch.FailedCounts = &failedCount
		batch.Update()
	}

	// Create output file record
	outputFileRecord := &model.BatchFile{
		UserId:      batch.UserId,
		TokenId:     batch.TokenId,
		Filename:    fmt.Sprintf("batch_output_%s.jsonl", batch.Id),
		Purpose:     model.FilePurposeBatchOutput,
		Bytes:       0, // Will be updated
		Status:      model.FileStatusProcessed,
		CreatedAt:   helper.GetTimestamp(),
		StoragePath: outputPath,
	}
	if err := outputFileRecord.Create(); err != nil {
		logger.SysErrorf("Failed to create output file record: %v", err)
	}

	// Get file size
	if stat, err := os.Stat(outputPath); err == nil {
		outputFileRecord.Bytes = stat.Size()
		outputFileRecord.Update()
	}

	// Update batch with output file
	batch.OutputFileId = &outputFileRecord.Id
	batch.SetStatus(model.BatchStatusCompleted)
}

// processBatchLine processes a single line from the batch input file
func processBatchLine(ctx context.Context, line string, lineNum int, batch *model.Batch) (string, error) {
	// Parse the batch request
	var batchReq struct {
		CustomId string          `json:"custom_id"`
		Method   string          `json:"method"`
		Url      string          `json:"url"`
		Body     json.RawMessage `json:"body"`
	}

	if err := json.Unmarshal([]byte(line), &batchReq); err != nil {
		return "", fmt.Errorf("invalid JSON at line %d: %v", lineNum, err)
	}

	// Get token for authentication
	token, err := model.GetTokenById(batch.TokenId)
	if err != nil {
		return "", fmt.Errorf("token not found: %v", err)
	}

	// Make the actual API call
	// This would typically call through the relay controller
	// For now, we'll return a placeholder response
	result := gin.H{
		"id":          fmt.Sprintf("batch_req_%d", lineNum),
		"custom_id":   batchReq.CustomId,
		"response": gin.H{
			"status_code": 200,
			"body": gin.H{
				"object": "chat.completion",
				"model":  "placeholder",
				"choices": []gin.H{
					{
						"index": 0,
						"message": gin.H{
							"role":    "assistant",
							"content": "Batch processing placeholder - integrate with relay controller",
						},
						"finish_reason": "stop",
					},
				},
			},
		},
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(resultBytes), nil
}