package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/random"
	"github.com/pagoda-inference/one-api/model"
)

type ProviderRequest struct {
	Code        string `json:"code"`
	Name        string `json:"name"`
	LogoUrl     string `json:"logo_url"`
	Description string `json:"description"`
	Website     string `json:"website"`
	Status      string `json:"status"`
	SortOrder   int    `json:"sort_order"`
}

// GetProviders 获取所有 Provider
func GetProviders(c *gin.Context) {
	providers, err := model.GetAllProviders()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    providers,
	})
}

// GetProvider 获取单个 Provider
func GetProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的 Provider ID"})
		return
	}
	p, err := model.GetProviderById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "Provider 不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": p})
}

// CreateProvider 创建 Provider
func CreateProvider(c *gin.Context) {
	var req ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	p := &model.Provider{
		Code:        req.Code,
		Name:        req.Name,
		LogoUrl:     req.LogoUrl,
		Description: req.Description,
		Website:     req.Website,
		Status:      req.Status,
		SortOrder:   req.SortOrder,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}

	if p.Code == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Provider Code 不能为空",
		})
		return
	}

	// Check if code already exists
	existing, _ := model.GetProviderByCode(p.Code)
	if existing != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Provider Code 已存在",
		})
		return
	}

	if p.Status == "" {
		p.Status = "active"
	}

	err := p.Create()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "创建失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    p,
	})
}

// UpdateProvider 更新 Provider
func UpdateProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的 Provider ID"})
		return
	}

	p, err := model.GetProviderById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Provider 不存在",
		})
		return
	}

	var req ProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	// 更新字段
	if req.Code != "" {
		p.Code = req.Code
	}
	if req.Name != "" {
		p.Name = req.Name
	}
	if req.LogoUrl != "" {
		p.LogoUrl = req.LogoUrl
	}
	if req.Description != "" {
		p.Description = req.Description
	}
	if req.Website != "" {
		p.Website = req.Website
	}
	if req.Status != "" {
		p.Status = req.Status
	}
	if req.SortOrder >= 0 {
		p.SortOrder = req.SortOrder
	}
	p.UpdatedAt = time.Now().Unix()

	err = p.Update()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "更新失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "更新成功",
		"data":    p,
	})
}

// DeleteProvider 删除 Provider
func DeleteProvider(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "无效的 Provider ID"})
		return
	}

	p, err := model.GetProviderById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "Provider 不存在",
		})
		return
	}

	// Check if provider has channels
	channels, _ := model.GetChannelsByProviderId(p.Code)
	if len(channels) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该 Provider 下有渠道，无法删除",
		})
		return
	}

	// Check if provider has models
	models, _ := model.GetModelsByProvider(p.Code)
	if len(models) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "该 Provider 下有模型，无法删除",
		})
		return
	}

	err = model.DeleteProvider(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "删除失败: " + err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// GetProviderStatuses 获取 Provider 状态列表
func GetProviderStatuses(c *gin.Context) {
	statuses := []gin.H{
		{"value": "active", "label": "正常"},
		{"value": "maintenance", "label": "维护中"},
		{"value": "disabled", "label": "禁用"},
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    statuses,
	})
}

// UploadProviderLogo 上传 Provider Logo
func UploadProviderLogo(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请选择文件",
		})
		return
	}

	// 验证文件类型
	ext := filepath.Ext(file.Filename)
	allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".svg": true, ".webp": true}
	if !allowedExts[ext] {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "不支持的文件格式，仅支持 jpg、png、svg、webp",
		})
		return
	}

	// 验证文件大小 (最大 2MB)
	if file.Size > 2*1024*1024 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "文件大小不能超过 2MB",
		})
		return
	}

	// 生成文件名
	filename := fmt.Sprintf("provider_%d_%s%s", time.Now().UnixNano(), random.GetUUID()[:8], ext)
	uploadDir := "/data/logos"

	// 确保目录存在
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "创建上传目录失败",
		})
		return
	}

	// 保存文件
	dst := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "保存文件失败: " + err.Error(),
		})
		return
	}

	url := "/logos/" + filename
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "上传成功",
		"data": gin.H{
			"url": url,
		},
	})
}

// ListLogos 获取已上传的 Logo 列表
func ListLogos(c *gin.Context) {
	uploadDir := "/data/logos"
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		// 目录不存在或无法读取，返回空列表
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data":    []gin.H{},
		})
		return
	}

	logos := make([]gin.H, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		ext := filepath.Ext(name)
		allowedExts := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".svg": true, ".webp": true}
		if !allowedExts[ext] {
			continue
		}
		logos = append(logos, gin.H{
			"name": name,
			"url":  "/logos/" + name,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    logos,
	})
}

// DeleteLogo 删除已上传的 Logo
func DeleteLogo(c *gin.Context) {
	filename := c.Query("filename")
	if filename == "" {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "文件名不能为空",
		})
		return
	}

	// 安全检查：只允许删除 logos 目录下的文件
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的文件名",
		})
		return
	}

	filepath := filepath.Join("/data/logos", filename)
	if err := os.Remove(filepath); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "删除失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "删除成功",
	})
}
