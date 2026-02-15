package http

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"small-merchant-ops-hub-server/internal/cache"
	"small-merchant-ops-hub-server/internal/db"
)

const summaryCacheKey = "merchant_ops:summary"

type createMemberRequest struct {
	Name    string `json:"name"`
	Phone   string `json:"phone"`
	Channel string `json:"channel"`
}

type createOrderRequest struct {
	OrderNo     string `json:"orderNo"`
	MemberID    uint   `json:"memberId"`
	AmountCents int64  `json:"amountCents"`
	Status      string `json:"status"`
	Source      string `json:"source"`
}

type memberResponse struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Phone     string    `json:"phone"`
	Channel   string    `json:"channel"`
	CreatedAt time.Time `json:"createdAt"`
}

type orderResponse struct {
	ID          uint       `json:"id"`
	OrderNo     string     `json:"orderNo"`
	MemberID    uint       `json:"memberId"`
	MemberName  string     `json:"memberName"`
	AmountCents int64      `json:"amountCents"`
	Status      string     `json:"status"`
	Source      string     `json:"source"`
	PaidAt      *time.Time `json:"paidAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type summaryResponse struct {
	MemberCount      int64             `json:"memberCount"`
	OrderCount       int64             `json:"orderCount"`
	PaidOrderCount   int64             `json:"paidOrderCount"`
	RevenueCents     int64             `json:"revenueCents"`
	RepurchaseCount  int64             `json:"repurchaseCount"`
	RepurchaseRate   float64           `json:"repurchaseRate"`
	ChannelBreakdown []channelResponse `json:"channelBreakdown"`
}

type channelResponse struct {
	Channel     string `json:"channel"`
	MemberCount int64  `json:"memberCount"`
}

func registerMerchantRoutes(router *gin.Engine, database *gorm.DB, cacheStore cache.Store) {
	api := router.Group("/api/v1")
	{
		api.GET("/members", listMembersHandler(database))
		api.POST("/members", createMemberHandler(database, cacheStore))

		api.GET("/orders", listOrdersHandler(database))
		api.POST("/orders", createOrderHandler(database, cacheStore))

		api.GET("/summary", summaryHandler(database, cacheStore))
	}
}

func createMemberHandler(database *gorm.DB, cacheStore cache.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createMemberRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, 400, "invalid member payload")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.Phone = strings.TrimSpace(req.Phone)
		req.Channel = strings.TrimSpace(req.Channel)

		if req.Name == "" || req.Phone == "" || req.Channel == "" {
			fail(c, 400, "name, phone and channel are required")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		member := db.Member{
			Name:    req.Name,
			Phone:   req.Phone,
			Channel: req.Channel,
		}
		if err := database.WithContext(ctx).Create(&member).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				fail(c, 400, "phone already exists")
				return
			}
			fail(c, 500, "create member failed")
			return
		}

		_ = cacheStore.Delete(ctx, summaryCacheKey)
		ok(c, toMemberResponse(member))
	}
}

func listMembersHandler(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		keyword := strings.TrimSpace(c.Query("q"))
		limit := parseLimit(c.Query("limit"), 20)

		query := database.WithContext(ctx).Model(&db.Member{}).Order("id DESC").Limit(limit)
		if keyword != "" {
			like := "%" + keyword + "%"
			query = query.Where("name LIKE ? OR phone LIKE ?", like, like)
		}

		members := make([]db.Member, 0, limit)
		if err := query.Find(&members).Error; err != nil {
			fail(c, 500, "list members failed")
			return
		}

		result := make([]memberResponse, 0, len(members))
		for _, member := range members {
			result = append(result, toMemberResponse(member))
		}

		ok(c, result)
	}
}

func createOrderHandler(database *gorm.DB, cacheStore cache.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createOrderRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, 400, "invalid order payload")
			return
		}

		req.Source = strings.TrimSpace(req.Source)
		req.Status = strings.TrimSpace(strings.ToLower(req.Status))
		req.OrderNo = strings.TrimSpace(req.OrderNo)

		if req.MemberID == 0 || req.AmountCents <= 0 || req.Source == "" {
			fail(c, 400, "memberId, amountCents and source are required")
			return
		}
		if req.Status == "" {
			req.Status = "paid"
		}
		if !isSupportedOrderStatus(req.Status) {
			fail(c, 400, "status must be pending, paid or refunded")
			return
		}
		if req.OrderNo == "" {
			req.OrderNo = generateOrderNo(req.MemberID)
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		var member db.Member
		if err := database.WithContext(ctx).First(&member, req.MemberID).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				fail(c, 400, "member not found")
				return
			}
			fail(c, 500, "query member failed")
			return
		}

		var paidAt *time.Time
		if req.Status == "paid" {
			now := time.Now()
			paidAt = &now
		}

		order := db.Order{
			OrderNo:     req.OrderNo,
			MemberID:    req.MemberID,
			AmountCents: req.AmountCents,
			Status:      req.Status,
			Source:      req.Source,
			PaidAt:      paidAt,
		}

		if err := database.WithContext(ctx).Create(&order).Error; err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "unique") {
				fail(c, 400, "orderNo already exists")
				return
			}
			fail(c, 500, "create order failed")
			return
		}

		_ = cacheStore.Delete(ctx, summaryCacheKey)

		ok(c, orderResponse{
			ID:          order.ID,
			OrderNo:     order.OrderNo,
			MemberID:    order.MemberID,
			MemberName:  member.Name,
			AmountCents: order.AmountCents,
			Status:      order.Status,
			Source:      order.Source,
			PaidAt:      order.PaidAt,
			CreatedAt:   order.CreatedAt,
		})
	}
}

func listOrdersHandler(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		limit := parseLimit(c.Query("limit"), 20)
		memberID := parseUint(c.Query("memberId"))

		query := database.WithContext(ctx).Model(&db.Order{}).Preload("Member").Order("id DESC").Limit(limit)
		if memberID > 0 {
			query = query.Where("member_id = ?", memberID)
		}

		orders := make([]db.Order, 0, limit)
		if err := query.Find(&orders).Error; err != nil {
			fail(c, 500, "list orders failed")
			return
		}

		result := make([]orderResponse, 0, len(orders))
		for _, order := range orders {
			result = append(result, orderResponse{
				ID:          order.ID,
				OrderNo:     order.OrderNo,
				MemberID:    order.MemberID,
				MemberName:  order.Member.Name,
				AmountCents: order.AmountCents,
				Status:      order.Status,
				Source:      order.Source,
				PaidAt:      order.PaidAt,
				CreatedAt:   order.CreatedAt,
			})
		}

		ok(c, result)
	}
}

func summaryHandler(database *gorm.DB, cacheStore cache.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if cached, found := getSummaryFromCache(ctx, cacheStore); found {
			ok(c, cached)
			return
		}

		var memberCount int64
		if err := database.WithContext(ctx).Model(&db.Member{}).Count(&memberCount).Error; err != nil {
			fail(c, 500, "count members failed")
			return
		}

		var orderCount int64
		if err := database.WithContext(ctx).Model(&db.Order{}).Count(&orderCount).Error; err != nil {
			fail(c, 500, "count orders failed")
			return
		}

		type paidAgg struct {
			PaidOrderCount int64 `gorm:"column:paid_order_count"`
			RevenueCents   int64 `gorm:"column:revenue_cents"`
		}
		var paid paidAgg
		if err := database.WithContext(ctx).
			Model(&db.Order{}).
			Select("COUNT(*) AS paid_order_count, COALESCE(SUM(amount_cents), 0) AS revenue_cents").
			Where("status = ?", "paid").
			Scan(&paid).Error; err != nil {
			fail(c, 500, "aggregate orders failed")
			return
		}

		sub := database.WithContext(ctx).
			Model(&db.Order{}).
			Select("member_id").
			Where("status = ?", "paid").
			Group("member_id").
			Having("COUNT(*) >= 2")

		var repurchaseCount int64
		if err := database.WithContext(ctx).Table("(?) AS repurchase_members", sub).Count(&repurchaseCount).Error; err != nil {
			fail(c, 500, "aggregate repurchase failed")
			return
		}

		type channelCount struct {
			Channel     string `gorm:"column:channel"`
			MemberCount int64  `gorm:"column:member_count"`
		}
		channelRows := make([]channelCount, 0)
		if err := database.WithContext(ctx).
			Model(&db.Member{}).
			Select("channel, COUNT(*) AS member_count").
			Group("channel").
			Order("member_count DESC").
			Scan(&channelRows).Error; err != nil {
			fail(c, 500, "aggregate channels failed")
			return
		}

		channelBreakdown := make([]channelResponse, 0, len(channelRows))
		for _, row := range channelRows {
			channelBreakdown = append(channelBreakdown, channelResponse{
				Channel:     row.Channel,
				MemberCount: row.MemberCount,
			})
		}

		repurchaseRate := 0.0
		if memberCount > 0 {
			repurchaseRate = math.Round((float64(repurchaseCount)/float64(memberCount))*10000) / 100
		}

		result := summaryResponse{
			MemberCount:      memberCount,
			OrderCount:       orderCount,
			PaidOrderCount:   paid.PaidOrderCount,
			RevenueCents:     paid.RevenueCents,
			RepurchaseCount:  repurchaseCount,
			RepurchaseRate:   repurchaseRate,
			ChannelBreakdown: channelBreakdown,
		}

		setSummaryToCache(ctx, cacheStore, result)
		ok(c, result)
	}
}

func toMemberResponse(member db.Member) memberResponse {
	return memberResponse{
		ID:        member.ID,
		Name:      member.Name,
		Phone:     member.Phone,
		Channel:   member.Channel,
		CreatedAt: member.CreatedAt,
	}
}

func parseLimit(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < 1 {
		return 1
	}
	if value > 100 {
		return 100
	}
	return value
}

func parseUint(raw string) uint {
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseUint(raw, 10, 64)
	if err != nil {
		return 0
	}
	return uint(value)
}

func isSupportedOrderStatus(status string) bool {
	switch status {
	case "pending", "paid", "refunded":
		return true
	default:
		return false
	}
}

func generateOrderNo(memberID uint) string {
	return fmt.Sprintf("ORD-%d-%d", memberID, time.Now().UnixNano())
}

func getSummaryFromCache(ctx context.Context, cacheStore cache.Store) (summaryResponse, bool) {
	raw, found, err := cacheStore.Get(ctx, summaryCacheKey)
	if err != nil || !found {
		return summaryResponse{}, false
	}
	var result summaryResponse
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		return summaryResponse{}, false
	}
	return result, true
}

func setSummaryToCache(ctx context.Context, cacheStore cache.Store, payload summaryResponse) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	_ = cacheStore.Set(ctx, summaryCacheKey, string(raw), 45*time.Second)
}
