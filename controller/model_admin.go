package controller

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/random"
	"github.com/pagoda-inference/one-api/model"
)

type ModelRequest struct {
	Id            string `json:"id"`
	Name          string `json:"name" binding:"required"`
	Provider      string `json:"provider"`
	ModelType     string `json:"model_type"`
	Description   string `json:"description"`
	ContextLen    int    `json:"context_len"`
	InputPrice    float64 `json:"input_price"`
	OutputPrice   float64 `json:"output_price"`
	Capabilities  string `json:"capabilities"`
	Status        string `json:"status"`
	IconUrl       string `json:"icon_url"`
	SortOrder     int    `json:"sort_order"`
}

type UploadLogoResponse struct {
	Url string `json:"url"`
}

// AdminListModels 获取所有模型
func AdminListModels(c *gin.Context) {
	models, err := model.GetAllAdminModels()
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
	id := c.Param("id")
	m, err := model.GetAdminModelById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "模型不存在",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    m,
	})
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

	m := &model.Model{
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
		CreatedTime:  time.Now().Unix(),
		UpdatedTime:  time.Now().Unix(),
	}

	if m.Id == "" {
		m.Id = random.GetUUID()
	}
	if m.Status == "" {
		m.Status = "active"
	}

	err := model.CreateModel(m)
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
		"data":    m,
	})
}

// UpdateModel 更新模型
func UpdateModel(c *gin.Context) {
	id := c.Param("id")
	var req ModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "参数错误: " + err.Error(),
		})
		return
	}

	m, err := model.GetAdminModelById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "模型不存在",
		})
		return
	}

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
		m.SortOrder = req.SortOrder
	}
	m.UpdatedTime = time.Now().Unix()

	err = model.UpdateModel(m)
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
	id := c.Param("id")
	err := model.DeleteModel(id)
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
	uploadDir := "web/public/logos"

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
