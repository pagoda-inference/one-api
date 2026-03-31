package model

import (
	"errors"
	"time"

	"github.com/pagoda-inference/one-api/common/helper"
	"github.com/pagoda-inference/one-api/common/random"
)

// BatchFile represents a file uploaded for batch processing
type BatchFile struct {
	Id             int    `json:"id" gorm:"primaryKey"`
	UserId         int    `json:"user_id" gorm:"index"`
	TokenId        int    `json:"token_id" gorm:"index"`
	Filename       string `json:"filename" gorm:"size:255"`
	Purpose        string `json:"purpose" gorm:"size:64"`  // "batch" or "batch_output"
	Bytes          int64  `json:"bytes"`
	Status         string `json:"status" gorm:"size:32"`   // "uploaded", "processed", "error"
	CreatedAt      int64  `json:"created_at" gorm:"bigint"`
	ExpiresAt      int64  `json:"expires_at" gorm:"bigint"` // Unix timestamp, 0 means no expiry
	StoragePath    string `json:"-" gorm:"size:512"`        // Internal storage path, not exposed to API
	ProcessError   string `json:"-" gorm:"type:text"`       // Error message if processing failed
}

// Batch represents a batch processing job
type Batch struct {
	Id                 string `json:"id" gorm:"primaryKey;size:32"` // batch-xxx format
	UserId             int    `json:"user_id" gorm:"index"`
	TokenId            int    `json:"token_id" gorm:"index"`
	Status             string `json:"status" gorm:"size:32;index"` // "validating", "in_progress", "finalizing", "completed", "expired", "cancelling", "cancelled", "failed"
	InputFileId        int    `json:"input_file_id"`
	OutputFileId       *int   `json:"output_file_id"`
	ErrorFileId        *int   `json:"error_file_id"`
	Endpoint           string `json:"endpoint" gorm:"size:128"`     // "/v1/chat/completions"
	CompletionWindow  string `json:"completion_window" gorm:"size:32"` // "24h"
	CreatedAt          int64  `json:"created_at" gorm:"bigint"`
	InProgressAt       *int64 `json:"in_progress_at" gorm:"bigint"`
	ExpiresAt          int64  `json:"expires_at" gorm:"bigint"`
	FinalizingAt       *int64 `json:"finalizing_at" gorm:"bigint"`
	CompletedAt        *int64 `json:"completed_at" gorm:"bigint"`
	CancelledAt        *int64 `json:"cancelled_at" gorm:"bigint"`
	FailedAt           *int64 `json:"failed_at" gorm:"bigint"`
	RequestCounts      *int   `json:"request_counts"`              // Total number of requests
	CompletedCounts    *int   `json:"completed_counts"`
	FailedCounts       *int   `json:"failed_counts"`
	Metadata           string `json:"metadata" gorm:"type:text"`   // JSON metadata
}

// Batch status constants
const (
	BatchStatusValidating  = "validating"
	BatchStatusInProgress  = "in_progress"
	BatchStatusFinalizing  = "finalizing"
	BatchStatusCompleted   = "completed"
	BatchStatusExpired     = "expired"
	BatchStatusCancelling  = "cancelling"
	BatchStatusCancelled   = "cancelled"
	BatchStatusFailed      = "failed"
)

// File status constants
const (
	FileStatusUploaded  = "uploaded"
	FileStatusProcessed = "processed"
	FileStatusError     = "error"
)

// File purpose constants
const (
	FilePurposeBatch       = "batch"
	FilePurposeBatchOutput = "batch_output"
)

// TableName for BatchFile
func (BatchFile) TableName() string {
	return "batch_files"
}

// TableName for Batch
func (Batch) TableName() string {
	return "batches"
}

// CreateFile creates a new batch file record
func (f *BatchFile) Create() error {
	if f.UserId == 0 {
		return errors.New("user_id is required")
	}
	if f.CreatedAt == 0 {
		f.CreatedAt = helper.GetTimestamp()
	}
	if f.Status == "" {
		f.Status = FileStatusUploaded
	}
	return DB.Create(f).Error
}

// GetFileById retrieves a file by ID
func GetFileById(id int) (*BatchFile, error) {
	var file BatchFile
	err := DB.First(&file, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetUserFileById retrieves a file by ID and user ID
func GetUserFileById(id int, userId int) (*BatchFile, error) {
	var file BatchFile
	err := DB.First(&file, "id = ? AND user_id = ?", id, userId).Error
	if err != nil {
		return nil, err
	}
	return &file, nil
}

// GetUserFiles lists files for a user
func GetUserFiles(userId int, purpose string, limit int, after int) ([]*BatchFile, error) {
	var files []*BatchFile
	query := DB.Where("user_id = ?", userId)
	if purpose != "" {
		query = query.Where("purpose = ?", purpose)
	}
	if after > 0 {
		query = query.Where("id < ?", after)
	}
	if limit <= 0 {
		limit = 100
	}
	err := query.Order("id DESC").Limit(limit).Find(&files).Error
	return files, err
}

// DeleteFile deletes a file
func (f *BatchFile) Delete() error {
	return DB.Delete(f).Error
}

// Update updates a file
func (f *BatchFile) Update() error {
	return DB.Save(f).Error
}

// Create creates a new batch job
func (b *Batch) Create() error {
	if b.UserId == 0 {
		return errors.New("user_id is required")
	}
	if b.Id == "" {
		b.Id = GenerateBatchId()
	}
	if b.CreatedAt == 0 {
		b.CreatedAt = helper.GetTimestamp()
	}
	if b.Status == "" {
		b.Status = BatchStatusValidating
	}
	return DB.Create(b).Error
}

// GenerateBatchId generates a unique batch ID
func GenerateBatchId() string {
	return "batch-" + random.GetRandomString(24)
}

// GetBatchById retrieves a batch by ID
func GetBatchById(id string) (*Batch, error) {
	var batch Batch
	err := DB.First(&batch, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// GetUserBatchById retrieves a batch by ID and user ID
func GetUserBatchById(id string, userId int) (*Batch, error) {
	var batch Batch
	err := DB.First(&batch, "id = ? AND user_id = ?", id, userId).Error
	if err != nil {
		return nil, err
	}
	return &batch, nil
}

// GetUserBatches lists batches for a user
func GetUserBatches(userId int, status string, limit int, after string) ([]*Batch, error) {
	var batches []*Batch
	query := DB.Where("user_id = ?", userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if after != "" {
		query = query.Where("id < ?", after)
	}
	if limit <= 0 {
		limit = 100
	}
	err := query.Order("id DESC").Limit(limit).Find(&batches).Error
	return batches, err
}

// Update updates a batch
func (b *Batch) Update() error {
	return DB.Save(b).Error
}

// Cancel cancels a batch job
func (b *Batch) Cancel() error {
	now := helper.GetTimestamp()
	b.Status = BatchStatusCancelled
	b.CancelledAt = &now
	return DB.Model(b).Updates(map[string]interface{}{
		"status":       BatchStatusCancelled,
		"cancelled_at": now,
	}).Error
}

// SetStatus sets the batch status with timestamp
func (b *Batch) SetStatus(status string) error {
	now := helper.GetTimestamp()
	updates := map[string]interface{}{
		"status": status,
	}

	switch status {
	case BatchStatusInProgress:
		b.InProgressAt = &now
		updates["in_progress_at"] = now
	case BatchStatusFinalizing:
		b.FinalizingAt = &now
		updates["finalizing_at"] = now
	case BatchStatusCompleted:
		b.CompletedAt = &now
		updates["completed_at"] = now
	case BatchStatusFailed:
		b.FailedAt = &now
		updates["failed_at"] = now
	}

	b.Status = status
	return DB.Model(b).Updates(updates).Error
}

// GetPendingBatches retrieves batches that need processing
func GetPendingBatches(limit int) ([]*Batch, error) {
	var batches []*Batch
	err := DB.Where("status IN ?", []string{BatchStatusValidating, BatchStatusInProgress}).
		Where("expires_at = 0 OR expires_at > ?", time.Now().Unix()).
		Order("created_at ASC").
		Limit(limit).
		Find(&batches).Error
	return batches, err
}

// CountUserBatches counts batches for a user
func CountUserBatches(userId int) (int64, error) {
	var count int64
	err := DB.Model(&Batch{}).Where("user_id = ?", userId).Count(&count).Error
	return count, err
}

// CountUserFiles counts files for a user
func CountUserFiles(userId int, purpose string) (int64, error) {
	var count int64
	query := DB.Model(&BatchFile{}).Where("user_id = ?", userId)
	if purpose != "" {
		query = query.Where("purpose = ?", purpose)
	}
	err := query.Count(&count).Error
	return count, err
}