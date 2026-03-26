package model

import (
	"errors"
	"time"

	"github.com/pagoda-inference/one-api/common/helper"
)

// TopupOrder represents a top-up order
type TopupOrder struct {
	Id          string  `json:"id" gorm:"primaryKey;size:32"`
	UserId      int     `json:"user_id" gorm:"index"`
	Amount      float64 `json:"amount" gorm:"type:decimal(10,2)"` // 充值金额(元)
	Quota       int64   `json:"quota" gorm:"default:0"`          // 获得额度
	Status      string  `json:"status" gorm:"size:32;index"`     // pending/paid/cancelled/refunded
	PayMethod   string  `json:"pay_method" gorm:"size:32"`       // alipay/wechat/card
	PayOrderId  string  `json:"pay_order_id" gorm:"size:64"`      // 第三方订单号
	CreatedAt   int64   `json:"created_at" gorm:"bigint"`
	PaidAt      *int64  `json:"paid_at" gorm:"bigint"`           // 支付完成时间
	ExpiredAt   int64   `json:"expired_at" gorm:"bigint"`        // 订单过期时间
	Description string  `json:"description" gorm:"size:255"`     // 描述
}

// Invoice represents an invoice application
type Invoice struct {
	Id          string  `json:"id" gorm:"primaryKey;size:32"`
	UserId      int     `json:"user_id" gorm:"index"`
	OrderIds    string  `json:"order_ids" gorm:"size:512"`      // 关联订单，逗号分隔
	Amount      float64 `json:"amount" gorm:"type:decimal(10,2)"` // 发票金额
	Status      string  `json:"status" gorm:"size:32;index"`     // pending/approved/issued/rejected
	Title       string  `json:"title" gorm:"size:255"`           // 发票抬头
	TaxNo       string  `json:"tax_no" gorm:"size:64"`          // 税号
	Address     string  `json:"address" gorm:"size:255"`         // 开票地址
	Phone       string  `json:"phone" gorm:"size:32"`             // 电话
	Bank        string  `json:"bank" gorm:"size:128"`            // 开户行
	Account     string  `json:"account" gorm:"size:64"`           // 银行账号
	Email       string  `json:"email" gorm:"size:128"`           // 接收邮箱
	Remark      string  `json:"remark" gorm:"size:512"`          // 备注
	CreatedAt   int64   `json:"created_at" gorm:"bigint"`
	ProcessedAt  *int64  `json:"processed_at" gorm:"bigint"`    // 处理时间
	IssuedAt    *int64   `json:"issued_at" gorm:"bigint"`        // 开票时间
}

// Order status constants
const (
	TopupStatusPending   = "pending"
	TopupStatusPaid      = "paid"
	TopupStatusCancelled = "cancelled"
	TopupStatusRefunded  = "refunded"
	TopupStatusExpired   = "expired"
)

// Invoice status constants
const (
	InvoiceStatusPending  = "pending"
	InvoiceStatusApproved = "approved"
	InvoiceStatusIssued   = "issued"
	InvoiceStatusRejected = "rejected"
)

// Pay method constants
const (
	PayMethodAlipay = "alipay"
	PayMethodWechat  = "wechat"
	PayMethodCard    = "card"
)

// TableName for TopupOrder
func (TopupOrder) TableName() string {
	return "topup_orders"
}

// TableName for Invoice
func (Invoice) TableName() string {
	return "invoices"
}

// GenerateOrderId generates a unique order ID
func GenerateOrderId(prefix string) string {
	timestamp := time.Now().UnixNano()
	return prefix + helper.GetRandomString(24-len(prefix)) + helper.FormatInt64(timestamp)[8:]
}

// CreateTopupOrder creates a new top-up order
func (t *TopupOrder) Create() error {
	if t.UserId == 0 {
		return errors.New("user_id is required")
	}
	if t.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if t.Id == "" {
		t.Id = GenerateOrderId("TU")
	}
	if t.Status == "" {
		t.Status = TopupStatusPending
	}
	if t.CreatedAt == 0 {
		t.CreatedAt = helper.GetTimestamp()
	}
	if t.ExpiredAt == 0 {
		t.ExpiredAt = t.CreatedAt + 30*60 // 30 minutes expiry
	}
	// Calculate quota based on amount (1元 = 1美元汇率 ≈ 7200 quota)
	// This can be made configurable
	t.Quota = int64(t.Amount * 7200)
	return DB.Create(t).Error
}

// GetTopupOrderById retrieves an order by ID
func GetTopupOrderById(id string) (*TopupOrder, error) {
	var order TopupOrder
	err := DB.First(&order, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetUserTopupOrderById retrieves an order by ID and user ID
func GetUserTopupOrderById(id string, userId int) (*TopupOrder, error) {
	var order TopupOrder
	err := DB.First(&order, "id = ? AND user_id = ?", id, userId).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

// GetUserTopupOrders lists top-up orders for a user
func GetUserTopupOrders(userId int, status string, limit int, offset int) ([]*TopupOrder, error) {
	var orders []*TopupOrder
	query := DB.Where("user_id = ?", userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if limit <= 0 {
		limit = 20
	}
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&orders).Error
	return orders, err
}

// UpdateTopupOrder updates a top-up order
func (t *TopupOrder) Update() error {
	return DB.Save(t).Error
}

// MarkOrderPaid marks an order as paid
func (t *TopupOrder) MarkOrderPaid(payOrderId string) error {
	now := helper.GetTimestamp()
	t.Status = TopupStatusPaid
	t.PayOrderId = payOrderId
	t.PaidAt = &now
	return DB.Model(t).Updates(map[string]interface{}{
		"status":       TopupStatusPaid,
		"pay_order_id": payOrderId,
		"paid_at":      now,
	}).Error
}

// CancelOrder cancels an order
func (t *TopupOrder) CancelOrder() error {
	t.Status = TopupStatusCancelled
	return DB.Model(t).Update("status", TopupStatusCancelled).Error
}

// IsExpired checks if the order is expired
func (t *TopupOrder) IsExpired() bool {
	if t.Status != TopupStatusPending {
		return false
	}
	return time.Now().Unix() > t.ExpiredAt
}

// CreateInvoice creates a new invoice application
func (i *Invoice) Create() error {
	if i.UserId == 0 {
		return errors.New("user_id is required")
	}
	if i.Title == "" {
		return errors.New("title is required")
	}
	if i.Amount <= 0 {
		return errors.New("amount must be positive")
	}
	if i.Id == "" {
		i.Id = GenerateOrderId("IV")
	}
	if i.Status == "" {
		i.Status = InvoiceStatusPending
	}
	if i.CreatedAt == 0 {
		i.CreatedAt = helper.GetTimestamp()
	}
	return DB.Create(i).Error
}

// GetInvoiceById retrieves an invoice by ID
func GetInvoiceById(id string) (*Invoice, error) {
	var invoice Invoice
	err := DB.First(&invoice, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

// GetUserInvoiceById retrieves an invoice by ID and user ID
func GetUserInvoiceById(id string, userId int) (*Invoice, error) {
	var invoice Invoice
	err := DB.First(&invoice, "id = ? AND user_id = ?", id, userId).Error
	if err != nil {
		return nil, err
	}
	return &invoice, nil
}

// GetUserInvoices lists invoice applications for a user
func GetUserInvoices(userId int, status string, limit int, offset int) ([]*Invoice, error) {
	var invoices []*Invoice
	query := DB.Where("user_id = ?", userId)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if limit <= 0 {
		limit = 20
	}
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&invoices).Error
	return invoices, err
}

// UpdateInvoice updates an invoice
func (i *Invoice) Update() error {
	return DB.Save(i).Error
}

// ApproveInvoice approves an invoice
func (i *Invoice) ApproveInvoice() error {
	now := helper.GetTimestamp()
	i.Status = InvoiceStatusApproved
	i.ProcessedAt = &now
	return DB.Model(i).Updates(map[string]interface{}{
		"status":       InvoiceStatusApproved,
		"processed_at": now,
	}).Error
}

// IssueInvoice marks an invoice as issued
func (i *Invoice) IssueInvoice() error {
	now := helper.GetTimestamp()
	i.Status = InvoiceStatusIssued
	i.IssuedAt = &now
	return DB.Model(i).Updates(map[string]interface{}{
		"status":     InvoiceStatusIssued,
		"issued_at":  now,
	}).Error
}

// RejectInvoice rejects an invoice
func (i *Invoice) RejectInvoice(reason string) error {
	now := helper.GetTimestamp()
	i.Status = InvoiceStatusRejected
	i.Remark = reason
	i.ProcessedAt = &now
	return DB.Model(i).Updates(map[string]interface{}{
		"status":       InvoiceStatusRejected,
		"remark":       reason,
		"processed_at":  now,
	}).Error
}

// CountUserTopupOrders counts top-up orders for a user
func CountUserTopupOrders(userId int) (int64, error) {
	var count int64
	err := DB.Model(&TopupOrder{}).Where("user_id = ?", userId).Count(&count).Error
	return count, err
}

// SumUserTopupAmount calculates total top-up amount for a user
func SumUserTopupAmount(userId int, status string) (float64, error) {
	var sum float64
	err := DB.Model(&TopupOrder{}).
		Where("user_id = ?", userId).
		Where("status = ?", status).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sum).Error
	return sum, err
}

// GetExpiredPendingOrders gets pending orders that have expired
func GetExpiredPendingOrders(limit int) ([]*TopupOrder, error) {
	var orders []*TopupOrder
	now := time.Now().Unix()
	err := DB.Where("status = ? AND expired_at < ?", TopupStatusPending, now).
		Limit(limit).
		Find(&orders).Error
	return orders, err
}