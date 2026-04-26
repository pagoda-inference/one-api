package controller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/i18n"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
)

var s3Client *s3.Client
var s3ClientInitOnce bool
var unsafeObjectKeyChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func normalizeObjectKey(raw string) string {
	key := strings.TrimSpace(raw)
	if key == "" {
		return ""
	}
	if unescaped, err := url.PathUnescape(key); err == nil && unescaped != "" {
		return unescaped
	}
	return key
}

func sanitizeObjectName(filename string) string {
	name := strings.TrimSpace(filepath.Base(filename))
	if name == "" {
		return "upload.bin"
	}
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	base = unsafeObjectKeyChars.ReplaceAllString(base, "-")
	base = strings.Trim(base, "-._")
	if base == "" {
		base = "upload"
	}
	ext = strings.ToLower(ext)
	ext = unsafeObjectKeyChars.ReplaceAllString(ext, "")
	return base + ext
}

func getS3Client() (*s3.Client, error) {
	if s3ClientInitOnce {
		return s3Client, nil
	}

	resolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
		return aws.Endpoint{
			URL:               fmt.Sprintf("http://%s", config.MinIOEndpoint),
			HostnameImmutable: true,
		}, nil
	})

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion("us-east-1"),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			config.MinIOAccessKey,
			config.MinIOSecretKey,
			"",
		)),
		awsconfig.WithEndpointResolverWithOptions(resolver),
	)
	if err != nil {
		return nil, err
	}

	s3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	s3ClientInitOnce = true
	return s3Client, nil
}

func UploadMediaFile(c *gin.Context) {
	if !config.MinIOEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "upload_not_enabled"),
		})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}
	defer file.Close()

	// Validate file size (max 10MB)
	if header.Size > 10*1024*1024 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "file_too_large"),
		})
		return
	}

	// Validate file type
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".svg": true, ".pdf": true, ".mp4": true, ".mov": true, ".webm": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "file_type_not_allowed"),
		})
		return
	}

	// Read file content
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "upload_failed"),
		})
		return
	}

	// Generate unique object key
	objectKey := fmt.Sprintf("%d-%s", time.Now().UnixMilli(), sanitizeObjectName(header.Filename))

	// Upload to MinIO
	client, err := getS3Client()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("s3_client_error: %s", err.Error()),
		})
		return
	}

	_, err = client.PutObject(c.Request.Context(), &s3.PutObjectInput{
		Bucket:      aws.String(config.MinIOBucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(header.Header.Get("Content-Type")),
	})
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("upload_failed: %s", err.Error()),
		})
		return
	}

	// Build URL on one-api's own domain (proxied through /api/images/)
	fileURL := fmt.Sprintf("/api/images/%s", url.PathEscape(objectKey))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"url":      fileURL,
			"key":      objectKey,
			"filename": header.Filename,
			"size":     header.Size,
		},
	})
}

func GetMediaFile(c *gin.Context) {
	if !config.MinIOEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "upload_not_enabled"),
		})
		return
	}

	key := normalizeObjectKey(c.Param("key"))
	if key == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}

	client, err := getS3Client()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("s3_client_error: %s", err.Error()),
		})
		return
	}

	result, err := client.GetObject(c.Request.Context(), &s3.GetObjectInput{
		Bucket: aws.String(config.MinIOBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}
	defer result.Body.Close()

	contentType := aws.ToString(result.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("Access-Control-Allow-Origin", "*")

	io.Copy(c.Writer, result.Body)
}

func HeadMediaFile(c *gin.Context) {
	if !config.MinIOEnabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "upload_not_enabled"),
		})
		return
	}

	key := normalizeObjectKey(c.Param("key"))
	if key == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": i18n.Translate(c, "invalid_parameter"),
		})
		return
	}

	client, err := getS3Client()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": fmt.Sprintf("s3_client_error: %s", err.Error()),
		})
		return
	}

	result, err := client.HeadObject(c.Request.Context(), &s3.HeadObjectInput{
		Bucket: aws.String(config.MinIOBucket),
		Key:    aws.String(key),
	})
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	contentType := aws.ToString(result.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	c.Header("Content-Type", contentType)
	if result.ContentLength != nil && *result.ContentLength > 0 {
		c.Header("Content-Length", fmt.Sprintf("%d", *result.ContentLength))
	}
	c.Header("Cache-Control", "public, max-age=31536000")
	c.Header("Access-Control-Allow-Origin", "*")
	c.Status(http.StatusOK)
}
