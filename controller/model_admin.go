package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/random"
	"github.com/pagoda-inference/one-api/model"
	"gorm.io/gorm"
)

type ModelRequest struct {
	Id            string  `json:"id"`
	Name          string  `json:"name"`
	Provider      string  `json:"provider"`
	ModelType     string  `json:"model_type"`
	Description   string  `json:"description"`
	ContextLen    int     `json:"context_len"`
	InputPrice    float64 `json:"input_price"`
	OutputPrice   float64 `json:"output_price"`
	Capabilities  string  `json:"capabilities"`
	Status        string  `json:"status"`
	IconUrl       string  `json:"icon_url"`
	SortOrder     int     `json:"sort_order"`
	RateLimitRPM  int     `json:"rate_limit_rpm"`
	RateLimitTPM  int     `json:"rate_limit_tpm"`
	VisibleToTeams string  `json:"visible_to_teams"`
	IsTrial        bool    `json:"is_trial"`
}

type UploadLogoResponse struct {
	Url string `json:"url"`
}

// AdminListModels 获取所有模型（从 model_info 读取）
func AdminListModels(c *gin.Context) {
	models, err := model.GetAllMarketModels()
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
		"data":    models,
	})
}

// GetModel 获取单个模型
func GetModel(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "缺少模型ID"})
		return
	}
	m, err := model.GetMarketModelById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "模型不存在"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": m})
}

// CreateModel 创建模型
func CreateModel(c *gin.Context) {
	var req ModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	m := &model.ModelInfo{
		Id:           req.Id,
		Name:         req.Name,
		Provider:     req.Provider,
		ModelType:    req.ModelType,
		Description:  req.Description,
		ContextLen:   req.ContextLen,
		InputPrice:   req.InputPrice,
		OutputPrice:  req.OutputPrice,
		Capabilities: req.Capabilities,
		Status:       req.Status,
		IconUrl:      req.IconUrl,
		SortOrder:    req.SortOrder,
		RateLimitRPM: req.RateLimitRPM,
		RateLimitTPM: req.RateLimitTPM,
		VisibleToTeams: req.VisibleToTeams,
		CreatedAt:    time.Now().Unix(),
		UpdatedAt:    time.Now().Unix(),
	}

	if m.Id == "" {
		m.Id = random.GetUUID()
	}
	if m.Status == "" {
		m.Status = "active"
	}

	// 自动分配 sort_order：如果未指定或为0，则分配最大值+1
	if m.SortOrder <= 0 {
		var maxOrder int
		model.DB.Model(&model.ModelInfo{}).Select("COALESCE(MAX(sort_order), 0)").Scan(&maxOrder)
		m.SortOrder = maxOrder + 1
	} else {
		// 指定了 sort_order > 0，需要将 >= 该位置的所有记录往后移动一位
		model.DB.Model(&model.ModelInfo{}).
			Where("sort_order >= ?", m.SortOrder).
			Update("sort_order", gorm.Expr("sort_order + 1"))
	}

	err := m.Create()
	if err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "23505") {
			errMsg = "该模型ID已存在，请换一个ID或留空自动生成"
		}
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "创建失败: " + errMsg,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "创建成功",
		"data":    m,
	})
}

// UpdateModel 更新模型
func UpdateModel(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "缺少模型ID"})
		return
	}
	var req ModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	m, err := model.GetMarketModelById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "模型不存在",
		})
		return
	}

	// 保存旧的 sort_order，用于后续判断是否需要重排
	oldSortOrder := m.SortOrder
	newSortOrder := oldSortOrder

	// 更新字段
	if req.Name != "" {
		m.Name = req.Name
	}
	if req.Provider != "" {
		m.Provider = req.Provider
	}
	if req.ModelType != "" {
		m.ModelType = req.ModelType
	}
	if req.Description != "" {
		m.Description = req.Description
	}
	if req.ContextLen > 0 {
		m.ContextLen = req.ContextLen
	}
	m.InputPrice = req.InputPrice
	m.OutputPrice = req.OutputPrice
	if req.Capabilities != "" {
		m.Capabilities = req.Capabilities
	}
	if req.Status != "" {
		m.Status = req.Status
	}
	if req.IconUrl != "" {
		m.IconUrl = req.IconUrl
	}
	if req.SortOrder >= 0 {
		newSortOrder = req.SortOrder
		m.SortOrder = req.SortOrder
	}
	if req.RateLimitRPM >= 0 {
		m.RateLimitRPM = req.RateLimitRPM
	}
	if req.RateLimitTPM >= 0 {
		m.RateLimitTPM = req.RateLimitTPM
	}
	if req.VisibleToTeams != "" {
		m.VisibleToTeams = req.VisibleToTeams
	}
	m.IsTrial = req.IsTrial
	m.UpdatedAt = time.Now().Unix()

	// 如果 sort_order 发生了变化，需要重排其他模型
	if newSortOrder != oldSortOrder && newSortOrder > 0 {
		if newSortOrder > oldSortOrder {
			// 将 >= oldSortOrder+1 且 <= newSortOrder 的模型往后移动一位
			model.DB.Model(&model.ModelInfo{}).
				Where("sort_order > ? AND sort_order <= ?", oldSortOrder, newSortOrder).
				Update("sort_order", gorm.Expr("sort_order - 1"))
		} else {
			// 将 >= newSortOrder 且 < oldSortOrder 的模型往前移动一位
			model.DB.Model(&model.ModelInfo{}).
				Where("sort_order >= ? AND sort_order < ?", newSortOrder, oldSortOrder).
				Update("sort_order", gorm.Expr("sort_order + 1"))
		}
	}

	err = m.Update()
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
		"data":    m,
	})
}

// DeleteModel 删除模型
func DeleteModel(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "缺少模型ID"})
		return
	}
	err := model.DeleteMarketModel(id)
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

// BatchDeleteModels 批量删除模型
func BatchDeleteModels(c *gin.Context) {
	var req struct {
		Ids []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || len(req.Ids) == 0 {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": "请选择要删除的模型"})
		return
	}
	for _, id := range req.Ids {
		model.DeleteMarketModel(id)
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": fmt.Sprintf("已删除 %d 个模型", len(req.Ids))})
}

// UploadModelLogo 上传模型 Logo
func UploadModelLogo(c *gin.Context) {
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
	filename := fmt.Sprintf("model_%d_%s%s", time.Now().UnixNano(), random.GetUUID()[:8], ext)
	uploadDir := "/data/logos"

	// 确保目录存在
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "创建上传目录失败",
		})
		return
	}

	filepath := filepath.Join(uploadDir, filename)
	if err := c.SaveUploadedFile(file, filepath); err != nil {
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

// GroupRequest for model group CRUD - DEPRECATED, use channels.group instead
// type GroupRequest struct {
// 	Id          int    `json:"id"`
// 	Name        string `json:"name" binding:"required"`
// 	Code        string `json:"code"`
// 	Description string `json:"description"`
// 	IconUrl     string `json:"icon_url"`
// 	SortOrder   int    `json:"sort_order"`
// 	Status      string `json:"status"`
// }

// AdminListModelGroups - DEPRECATED
// func AdminListModelGroups(c *gin.Context) { ... }

// CreateModelGroup - DEPRECATED
// func CreateModelGroup(c *gin.Context) { ... }

// UpdateModelGroup - DEPRECATED
// func UpdateModelGroup(c *gin.Context) { ... }

// DeleteModelGroup - DEPRECATED
// func DeleteModelGroup(c *gin.Context) { ... }

// GetModelTypes 获取模型类型列表
func GetModelTypes(c *gin.Context) {
	types := []gin.H{
		{"value": "chat", "label": "对话模型"},
		{"value": "vlm", "label": "视觉模型"},
		{"value": "embedding", "label": "Embedding"},
		{"value": "reranker", "label": "Reranker"},
		{"value": "ocr", "label": "OCR"},
		{"value": "other", "label": "其他"},
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    types,
	})
}

// GetModelStatuses 获取模型状态列表
func GetModelStatuses(c *gin.Context) {
	statuses := []gin.H{
		{"value": "active", "label": "上线"},
		{"value": "maintenance", "label": "维护中"},
		{"value": "disabled", "label": "下架"},
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    statuses,
	})
}

// 权限检查中间件
func requireAdmin(c *gin.Context) {
	role := c.GetInt(ctxkey.Role)
	if role < 1 { // 0=普通用户, 1=管理员
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "需要管理员权限",
		})
		c.Abort()
		return
	}
	c.Next()
}

// MustAdmin 是 requireAdmin 的别名，用于兼容
var MustAdmin = requireAdmin
