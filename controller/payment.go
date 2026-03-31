package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/pagoda-inference/one-api/common/config"
	"github.com/pagoda-inference/one-api/common/ctxkey"
	"github.com/pagoda-inference/one-api/common/logger"
	"github.com/pagoda-inference/one-api/model"
)

// TopupRequest represents a top-up request
type TopupRequest struct {
	Amount    float64 `json:"amount" binding:"required,gt=0"`
	PayMethod string  `json:"pay_method" binding:"required"`
}

// CreateTopupOrder handles POST /api/topup
func CreateTopupOrder(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	var req TopupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// Validate pay method
	if req.PayMethod != model.PayMethodAlipay &&
		req.PayMethod != model.PayMethodWechat &&
		req.PayMethod != model.PayMethodCard {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid pay method",
		})
		return
	}

	// Check if payment is enabled
	if !config.PaymentEnabled {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"success": false,
			"message": "Payment service is disabled",
		})
		return
	}

	// Create order
	order := &model.TopupOrder{
		UserId:      userId,
		Amount:      req.Amount,
		PayMethod:   req.PayMethod,
		Status:      model.TopupStatusPending,
		Description: fmt.Sprintf("充值 %.2f 元", req.Amount),
	}

	if err := order.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create order: " + err.Error(),
		})
		return
	}

	// Generate payment URL based on pay method
	paymentUrl, err := generatePaymentUrl(order)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to generate payment: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"order_id":    order.Id,
			"amount":      order.Amount,
			"quota":       order.Quota,
			"pay_method":  order.PayMethod,
			"status":      order.Status,
			"payment_url": paymentUrl,
			"expired_at":  order.ExpiredAt,
		},
	})
}

// GetTopupOrders handles GET /api/topup
func GetTopupOrders(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	status := c.Query("status")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	orders, err := model.GetUserTopupOrders(userId, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get orders: " + err.Error(),
		})
		return
	}

	data := make([]gin.H, len(orders))
	for i, order := range orders {
		data[i] = formatTopupOrder(order)
	}

	total, _ := model.CountUserTopupOrders(userId)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"orders": data,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}

// GetTopupOrder handles GET /api/topup/:id
func GetTopupOrder(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	orderId := c.Param("id")

	order, err := model.GetUserTopupOrderById(orderId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Order not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    formatTopupOrder(order),
	})
}

// CancelTopupOrder handles POST /api/topup/:id/cancel
func CancelTopupOrder(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	orderId := c.Param("id")

	order, err := model.GetUserTopupOrderById(orderId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Order not found",
		})
		return
	}

	if order.Status != model.TopupStatusPending {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Only pending orders can be cancelled",
		})
		return
	}

	if err := order.CancelOrder(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to cancel order: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    formatTopupOrder(order),
	})
}

// AlipayNotify handles Alipay callback notification
func AlipayNotify(c *gin.Context) {
	if !config.AlipayEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Alipay not enabled"})
		return
	}

	// In production, verify the signature here
	outTradeNo := c.PostForm("out_trade_no")
	tradeStatus := c.PostForm("trade_status")
	tradeNo := c.PostForm("trade_no")

	logger.SysLogf("Alipay callback: out_trade_no=%s, trade_status=%s", outTradeNo, tradeStatus)

	order, err := model.GetTopupOrderById(outTradeNo)
	if err != nil {
		logger.SysErrorf("Alipay callback: order not found %s", outTradeNo)
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if tradeStatus == "TRADE_SUCCESS" || tradeStatus == "TRADE_FINISHED" {
		if err := handlePaymentSuccess(order, tradeNo); err != nil {
			logger.SysErrorf("Alipay callback: failed to process %s, error: %v", outTradeNo, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": "success"})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": "waiting"})
	}
}

// WechatPayNotify handles WeChat Pay callback notification
func WechatPayNotify(c *gin.Context) {
	if !config.WechatPayEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "WeChat Pay not enabled"})
		return
	}

	// In production, decrypt and verify the notification here
	// For now, we'll use a simplified version
	outTradeNo := c.PostForm("out_trade_no")
	tradeState := c.PostForm("trade_state")
	transactionId := c.PostForm("transaction_id")

	logger.SysLogf("WeChat Pay callback: out_trade_no=%s, trade_state=%s", outTradeNo, tradeState)

	order, err := model.GetTopupOrderById(outTradeNo)
	if err != nil {
		logger.SysErrorf("WeChat Pay callback: order not found %s", outTradeNo)
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	if tradeState == "SUCCESS" {
		if err := handlePaymentSuccess(order, transactionId); err != nil {
			logger.SysErrorf("WeChat Pay callback: failed to process %s, error: %v", outTradeNo, err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": "SUCCESS"})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": "FAIL"})
	}
}

// InvoiceRequest represents an invoice application request
type InvoiceRequest struct {
	OrderIds string  `json:"order_ids" binding:"required"`
	Amount   float64 `json:"amount" binding:"required,gt=0"`
	Title    string  `json:"title" binding:"required"`
	TaxNo    string  `json:"tax_no"`
	Address  string  `json:"address"`
	Phone    string  `json:"phone"`
	Bank     string  `json:"bank"`
	Account  string  `json:"account"`
	Email    string  `json:"email"`
	Remark   string  `json:"remark"`
}

// CreateInvoice handles POST /api/invoice
func CreateInvoice(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	var req InvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invalid request: " + err.Error(),
		})
		return
	}

	// Verify order IDs belong to user and are paid
	orderIdList := strings.Split(req.OrderIds, ",")
	totalAmount := 0.0
	for _, orderId := range orderIdList {
		orderId = strings.TrimSpace(orderId)
		order, err := model.GetUserTopupOrderById(orderId, userId)
		if err != nil || order.Status != model.TopupStatusPaid {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"message": fmt.Sprintf("Order %s is not valid or not paid", orderId),
			})
			return
		}
		totalAmount += order.Amount
	}

	// Check if amount matches
	if totalAmount < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "Invoice amount exceeds paid amount",
		})
		return
	}

	invoice := &model.Invoice{
		UserId:    userId,
		OrderIds:  req.OrderIds,
		Amount:    req.Amount,
		Title:     req.Title,
		TaxNo:     req.TaxNo,
		Address:   req.Address,
		Phone:     req.Phone,
		Bank:      req.Bank,
		Account:   req.Account,
		Email:     req.Email,
		Remark:    req.Remark,
		Status:    model.InvoiceStatusPending,
	}

	if err := invoice.Create(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to create invoice: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":         invoice.Id,
			"status":     invoice.Status,
			"created_at": invoice.CreatedAt,
		},
	})
}

// GetInvoices handles GET /api/invoice
func GetInvoices(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	status := c.Query("status")
	limit, _ := strconv.Atoi(c.Query("limit"))
	offset, _ := strconv.Atoi(c.Query("offset"))

	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	invoices, err := model.GetUserInvoices(userId, status, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": "Failed to get invoices: " + err.Error(),
		})
		return
	}

	data := make([]gin.H, len(invoices))
	for i, inv := range invoices {
		data[i] = formatInvoice(inv)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    data,
	})
}

// GetInvoice handles GET /api/invoice/:id
func GetInvoice(c *gin.Context) {
	userId := c.GetInt(ctxkey.Id)
	if userId == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{
			"success": false,
			"message": "Unauthorized",
		})
		return
	}

	invoiceId := c.Param("id")

	invoice, err := model.GetUserInvoiceById(invoiceId, userId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "Invoice not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    formatInvoice(invoice),
	})
}

// Helper functions

func formatTopupOrder(order *model.TopupOrder) gin.H {
	return gin.H{
		"id":          order.Id,
		"amount":      order.Amount,
		"quota":       order.Quota,
		"status":      order.Status,
		"pay_method":  order.PayMethod,
		"pay_order_id": order.PayOrderId,
		"created_at":  order.CreatedAt,
		"paid_at":     order.PaidAt,
		"expired_at":  order.ExpiredAt,
	}
}

func formatInvoice(inv *model.Invoice) gin.H {
	return gin.H{
		"id":          inv.Id,
		"order_ids":   inv.OrderIds,
		"amount":      inv.Amount,
		"status":      inv.Status,
		"title":       inv.Title,
		"tax_no":      inv.TaxNo,
		"address":     inv.Address,
		"phone":       inv.Phone,
		"bank":        inv.Bank,
		"account":     inv.Account,
		"email":       inv.Email,
		"remark":      inv.Remark,
		"created_at":  inv.CreatedAt,
		"processed_at": inv.ProcessedAt,
		"issued_at":   inv.IssuedAt,
	}
}

func generatePaymentUrl(order *model.TopupOrder) (string, error) {
	switch order.PayMethod {
	case model.PayMethodAlipay:
		if !config.AlipayEnabled {
			return "", fmt.Errorf("Alipay is not enabled")
		}
		// In production, generate Alipay payment URL
		return fmt.Sprintf("https://openapi.alipay.com/gateway.do?out_trade_no=%s&total_amount=%.2f", order.Id, order.Amount), nil

	case model.PayMethodWechat:
		if !config.WechatPayEnabled {
			return "", fmt.Errorf("WeChat Pay is not enabled")
		}
		// In production, generate WeChat Pay QR code URL
		return fmt.Sprintf("https://api.mch.weixin.qq.com/pay/unifiedorder?out_trade_no=%s&total_fee=%d", order.Id, int(order.Amount*100)), nil

	default:
		return "", fmt.Errorf("unsupported pay method")
	}
}

func handlePaymentSuccess(order *model.TopupOrder, payOrderId string) error {
	if order.Status == model.TopupStatusPaid {
		return nil // Already processed
	}

	if err := order.MarkOrderPaid(payOrderId); err != nil {
		return err
	}

	// Add quota to user
	user, err := model.GetUserById(order.UserId, true)
	if err != nil {
		return err
	}

	user.Quota += order.Quota
	if err := user.Update(true); err != nil {
		return err
	}

	// Record top-up log
	model.RecordTopupLog(nil, order.UserId, fmt.Sprintf("充值 %.2f 元，获得 %d quota", order.Amount, order.Quota), int(order.Quota))

	logger.SysLogf("Topup successful: order=%s, user=%d, amount=%.2f, quota=%d", order.Id, order.UserId, order.Amount, order.Quota)
	return nil
}

// CleanupExpiredOrders cleans up expired pending orders
func CleanupExpiredOrders() {
	orders, err := model.GetExpiredPendingOrders(100)
	if err != nil {
		logger.SysErrorf("Failed to get expired orders: %v", err)
		return
	}

	for _, order := range orders {
		if err := order.CancelOrder(); err != nil {
			logger.SysErrorf("Failed to cancel expired order %s: %v", order.Id, err)
		} else {
			logger.SysLogf("Cancelled expired order: %s", order.Id)
		}
	}
}

// StartPaymentCleanupWorker starts a background worker to clean up expired orders
func StartPaymentCleanupWorker() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			CleanupExpiredOrders()
		}
	}()
	logger.SysLog("Payment cleanup worker started")
}