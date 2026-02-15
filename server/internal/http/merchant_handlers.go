package http

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	stdhttp "net/http"
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

type createCampaignRequest struct {
	Name        string  `json:"name"`
	Channel     string  `json:"channel"`
	DiscountPct float64 `json:"discountPct"`
	Status      string  `json:"status"`
	StartAt     string  `json:"startAt"`
	EndAt       string  `json:"endAt"`
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

type campaignResponse struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Channel     string     `json:"channel"`
	DiscountPct float64    `json:"discountPct"`
	Status      string     `json:"status"`
	StartAt     *time.Time `json:"startAt"`
	EndAt       *time.Time `json:"endAt"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type followupResponse struct {
	DaysWindow int                    `json:"daysWindow"`
	Items      []followupMemberResult `json:"items"`
}

type followupMemberResult struct {
	MemberID         uint       `json:"memberId"`
	MemberName       string     `json:"memberName"`
	Phone            string     `json:"phone"`
	Channel          string     `json:"channel"`
	PaidOrderCount   int64      `json:"paidOrderCount"`
	PaidAmountCents  int64      `json:"paidAmountCents"`
	LastPaidAt       *time.Time `json:"lastPaidAt"`
	DaysSinceLastPay int        `json:"daysSinceLastPay"`
}

type summaryResponse struct {
	MemberCount         int64             `json:"memberCount"`
	OrderCount          int64             `json:"orderCount"`
	PaidOrderCount      int64             `json:"paidOrderCount"`
	RevenueCents        int64             `json:"revenueCents"`
	RepurchaseCount     int64             `json:"repurchaseCount"`
	RepurchaseRate      float64           `json:"repurchaseRate"`
	ActiveCampaignCount int64             `json:"activeCampaignCount"`
	ChannelBreakdown    []channelResponse `json:"channelBreakdown"`
}

type channelResponse struct {
	Channel     string `json:"channel"`
	MemberCount int64  `json:"memberCount"`
}

type campaignAttributionRow struct {
	CampaignID               uint       `json:"campaignId"`
	CampaignName             string     `json:"campaignName"`
	Channel                  string     `json:"channel"`
	Status                   string     `json:"status"`
	StartAt                  *time.Time `json:"startAt"`
	EndAt                    *time.Time `json:"endAt"`
	TargetMemberCount        int64      `json:"targetMemberCount"`
	PaidOrderCount           int64      `json:"paidOrderCount"`
	ConvertedMemberCount     int64      `json:"convertedMemberCount"`
	RepurchaseConvertedCount int64      `json:"repurchaseConvertedCount"`
	RevenueCents             int64      `json:"revenueCents"`
	ConversionRate           float64    `json:"conversionRate"`
}

type campaignAttributionPayload struct {
	Rows []campaignAttributionRow `json:"rows"`
}

func registerMerchantRoutes(router *gin.Engine, database *gorm.DB, cacheStore cache.Store) {
	api := router.Group("/api/v1")
	{
		api.GET("/members", listMembersHandler(database))
		api.POST("/members", createMemberHandler(database, cacheStore))

		api.GET("/orders", listOrdersHandler(database))
		api.POST("/orders", createOrderHandler(database, cacheStore))

		api.GET("/campaigns", listCampaignsHandler(database))
		api.POST("/campaigns", createCampaignHandler(database, cacheStore))

		api.GET("/followups", listFollowupsHandler(database))
		api.GET("/reports/campaign-attribution", campaignAttributionHandler(database))
		api.GET("/reports/campaign-attribution/export", campaignAttributionCSVHandler(database))
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
		ok(c, toOrderResponse(order, member.Name))
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
			result = append(result, toOrderResponse(order, order.Member.Name))
		}

		ok(c, result)
	}
}

func createCampaignHandler(database *gorm.DB, cacheStore cache.Store) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req createCampaignRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, 400, "invalid campaign payload")
			return
		}

		req.Name = strings.TrimSpace(req.Name)
		req.Channel = strings.TrimSpace(req.Channel)
		req.Status = strings.TrimSpace(strings.ToLower(req.Status))

		if req.Name == "" || req.Channel == "" {
			fail(c, 400, "name and channel are required")
			return
		}
		if req.DiscountPct <= 0 || req.DiscountPct > 100 {
			fail(c, 400, "discountPct must be in (0, 100]")
			return
		}
		if req.Status == "" {
			req.Status = "active"
		}
		if !isSupportedCampaignStatus(req.Status) {
			fail(c, 400, "status must be draft, active or closed")
			return
		}

		startAt, err := parseOptionalRFC3339(req.StartAt)
		if err != nil {
			fail(c, 400, "startAt must be RFC3339 format")
			return
		}
		endAt, err := parseOptionalRFC3339(req.EndAt)
		if err != nil {
			fail(c, 400, "endAt must be RFC3339 format")
			return
		}
		if startAt != nil && endAt != nil && endAt.Before(*startAt) {
			fail(c, 400, "endAt cannot be earlier than startAt")
			return
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		campaign := db.Campaign{
			Name:        req.Name,
			Channel:     req.Channel,
			DiscountPct: req.DiscountPct,
			Status:      req.Status,
			StartAt:     startAt,
			EndAt:       endAt,
		}
		if err := database.WithContext(ctx).Create(&campaign).Error; err != nil {
			fail(c, 500, "create campaign failed")
			return
		}

		_ = cacheStore.Delete(ctx, summaryCacheKey)
		ok(c, toCampaignResponse(campaign))
	}
}

func listCampaignsHandler(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		limit := parseLimit(c.Query("limit"), 20)
		status := strings.TrimSpace(strings.ToLower(c.Query("status")))
		channel := strings.TrimSpace(c.Query("channel"))

		query := database.WithContext(ctx).Model(&db.Campaign{}).Order("id DESC").Limit(limit)
		if status != "" {
			query = query.Where("status = ?", status)
		}
		if channel != "" {
			query = query.Where("channel = ?", channel)
		}

		campaigns := make([]db.Campaign, 0, limit)
		if err := query.Find(&campaigns).Error; err != nil {
			fail(c, 500, "list campaigns failed")
			return
		}

		result := make([]campaignResponse, 0, len(campaigns))
		for _, campaign := range campaigns {
			result = append(result, toCampaignResponse(campaign))
		}
		ok(c, result)
	}
}

func listFollowupsHandler(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		days := parseDays(c.Query("days"), 30)
		limit := parseLimit(c.Query("limit"), 50)
		channel := strings.TrimSpace(c.Query("channel"))
		cutoff := time.Now().AddDate(0, 0, -days)

		type followupRow struct {
			MemberID        uint   `gorm:"column:member_id"`
			MemberName      string `gorm:"column:member_name"`
			Phone           string `gorm:"column:phone"`
			Channel         string `gorm:"column:channel"`
			PaidOrderCount  int64  `gorm:"column:paid_order_count"`
			PaidAmountCents int64  `gorm:"column:paid_amount_cents"`
			LastPaidUnix    int64  `gorm:"column:last_paid_unix"`
		}

		rows := make([]followupRow, 0, limit)
		query := database.WithContext(ctx).
			Table("members AS m").
			Select(`
				m.id AS member_id,
				m.name AS member_name,
				m.phone AS phone,
				m.channel AS channel,
				COUNT(o.id) AS paid_order_count,
				COALESCE(SUM(o.amount_cents), 0) AS paid_amount_cents,
				MAX(CAST(strftime('%s', o.paid_at) AS INTEGER)) AS last_paid_unix
			`).
			Joins("LEFT JOIN orders AS o ON o.member_id = m.id AND o.status = ?", "paid").
			Group("m.id, m.name, m.phone, m.channel").
			Having("COUNT(o.id) = 1 OR MAX(CAST(strftime('%s', o.paid_at) AS INTEGER)) <= ?", cutoff.Unix()).
			Order("MAX(CAST(strftime('%s', o.paid_at) AS INTEGER)) ASC").
			Limit(limit)

		if channel != "" {
			query = query.Where("m.channel = ?", channel)
		}

		if err := query.Scan(&rows).Error; err != nil {
			fail(c, 500, "list followups failed")
			return
		}

		items := make([]followupMemberResult, 0, len(rows))
		for _, row := range rows {
			var lastPaidAt *time.Time
			if row.LastPaidUnix > 0 {
				value := time.Unix(row.LastPaidUnix, 0)
				lastPaidAt = &value
			}

			daysSinceLastPay := 0
			if lastPaidAt != nil {
				daysSinceLastPay = int(time.Since(*lastPaidAt).Hours() / 24)
				if daysSinceLastPay < 0 {
					daysSinceLastPay = 0
				}
			}
			items = append(items, followupMemberResult{
				MemberID:         row.MemberID,
				MemberName:       row.MemberName,
				Phone:            row.Phone,
				Channel:          row.Channel,
				PaidOrderCount:   row.PaidOrderCount,
				PaidAmountCents:  row.PaidAmountCents,
				LastPaidAt:       lastPaidAt,
				DaysSinceLastPay: daysSinceLastPay,
			})
		}

		ok(c, followupResponse{
			DaysWindow: days,
			Items:      items,
		})
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

		var activeCampaignCount int64
		if err := database.WithContext(ctx).Model(&db.Campaign{}).Where("status = ?", "active").Count(&activeCampaignCount).Error; err != nil {
			fail(c, 500, "count active campaigns failed")
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
			MemberCount:         memberCount,
			OrderCount:          orderCount,
			PaidOrderCount:      paid.PaidOrderCount,
			RevenueCents:        paid.RevenueCents,
			RepurchaseCount:     repurchaseCount,
			RepurchaseRate:      repurchaseRate,
			ActiveCampaignCount: activeCampaignCount,
			ChannelBreakdown:    channelBreakdown,
		}

		setSummaryToCache(ctx, cacheStore, result)
		ok(c, result)
	}
}

func campaignAttributionHandler(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		rows, err := loadCampaignAttributionRows(ctx, database, c)
		if err != nil {
			fail(c, 500, err.Error())
			return
		}
		ok(c, campaignAttributionPayload{
			Rows: rows,
		})
	}
}

func campaignAttributionCSVHandler(database *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()

		rows, err := loadCampaignAttributionRows(ctx, database, c)
		if err != nil {
			fail(c, 500, err.Error())
			return
		}

		content, err := buildCampaignAttributionCSV(rows)
		if err != nil {
			fail(c, 500, "build csv failed")
			return
		}

		c.Header("Content-Type", "text/csv; charset=utf-8")
		c.Header("Content-Disposition", "attachment; filename=campaign-attribution.csv")
		c.String(stdhttp.StatusOK, content)
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

func toOrderResponse(order db.Order, memberName string) orderResponse {
	return orderResponse{
		ID:          order.ID,
		OrderNo:     order.OrderNo,
		MemberID:    order.MemberID,
		MemberName:  memberName,
		AmountCents: order.AmountCents,
		Status:      order.Status,
		Source:      order.Source,
		PaidAt:      order.PaidAt,
		CreatedAt:   order.CreatedAt,
	}
}

func toCampaignResponse(campaign db.Campaign) campaignResponse {
	return campaignResponse{
		ID:          campaign.ID,
		Name:        campaign.Name,
		Channel:     campaign.Channel,
		DiscountPct: campaign.DiscountPct,
		Status:      campaign.Status,
		StartAt:     campaign.StartAt,
		EndAt:       campaign.EndAt,
		CreatedAt:   campaign.CreatedAt,
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

func parseDays(raw string, fallback int) int {
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
	if value > 365 {
		return 365
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

func parseOptionalRFC3339(raw string) (*time.Time, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	value, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func isSupportedOrderStatus(status string) bool {
	switch status {
	case "pending", "paid", "refunded":
		return true
	default:
		return false
	}
}

func isSupportedCampaignStatus(status string) bool {
	switch status {
	case "draft", "active", "closed":
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

func loadCampaignAttributionRows(
	ctx context.Context,
	database *gorm.DB,
	c *gin.Context,
) ([]campaignAttributionRow, error) {
	limit := parseLimit(c.Query("limit"), 100)
	status := strings.TrimSpace(strings.ToLower(c.Query("status")))
	channel := strings.TrimSpace(c.Query("channel"))
	keyword := strings.TrimSpace(c.Query("q"))

	from, err := parseOptionalRFC3339(c.Query("from"))
	if err != nil {
		return nil, fmt.Errorf("from must be RFC3339 format")
	}
	to, err := parseOptionalRFC3339(c.Query("to"))
	if err != nil {
		return nil, fmt.Errorf("to must be RFC3339 format")
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, fmt.Errorf("to cannot be earlier than from")
	}

	query := database.WithContext(ctx).Model(&db.Campaign{}).Order("id DESC").Limit(limit)
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if channel != "" {
		query = query.Where("channel = ?", channel)
	}
	if keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword+"%")
	}
	if from != nil {
		query = query.Where("created_at >= ?", *from)
	}
	if to != nil {
		query = query.Where("created_at <= ?", *to)
	}

	campaigns := make([]db.Campaign, 0, limit)
	if err := query.Find(&campaigns).Error; err != nil {
		return nil, fmt.Errorf("list campaigns failed")
	}

	repurchaseMembersSubQuery := database.WithContext(ctx).
		Model(&db.Order{}).
		Select("member_id").
		Where("status = ?", "paid").
		Group("member_id").
		Having("COUNT(*) >= 2")

	rows := make([]campaignAttributionRow, 0, len(campaigns))
	for _, campaign := range campaigns {
		var targetMemberCount int64
		if err := database.WithContext(ctx).
			Model(&db.Member{}).
			Where("channel = ?", campaign.Channel).
			Count(&targetMemberCount).Error; err != nil {
			return nil, fmt.Errorf("count target members failed")
		}

		orderScope := database.WithContext(ctx).
			Model(&db.Order{}).
			Where("status = ? AND source = ?", "paid", campaign.Channel)
		if campaign.StartAt != nil {
			orderScope = orderScope.Where("paid_at >= ?", *campaign.StartAt)
		}
		if campaign.EndAt != nil {
			orderScope = orderScope.Where("paid_at <= ?", *campaign.EndAt)
		}

		var paidOrderCount int64
		if err := orderScope.Count(&paidOrderCount).Error; err != nil {
			return nil, fmt.Errorf("count paid orders failed")
		}

		type revenueAgg struct {
			RevenueCents int64 `gorm:"column:revenue_cents"`
		}
		var revenue revenueAgg
		if err := orderScope.Select("COALESCE(SUM(amount_cents), 0) AS revenue_cents").Scan(&revenue).Error; err != nil {
			return nil, fmt.Errorf("aggregate revenue failed")
		}

		var convertedMemberCount int64
		if err := orderScope.Distinct("member_id").Count(&convertedMemberCount).Error; err != nil {
			return nil, fmt.Errorf("count converted members failed")
		}

		var repurchaseConvertedCount int64
		if err := orderScope.
			Where("member_id IN (?)", repurchaseMembersSubQuery).
			Distinct("member_id").
			Count(&repurchaseConvertedCount).Error; err != nil {
			return nil, fmt.Errorf("count repurchase converted members failed")
		}

		conversionRate := 0.0
		if targetMemberCount > 0 {
			conversionRate = math.Round((float64(convertedMemberCount)/float64(targetMemberCount))*10000) / 100
		}

		rows = append(rows, campaignAttributionRow{
			CampaignID:               campaign.ID,
			CampaignName:             campaign.Name,
			Channel:                  campaign.Channel,
			Status:                   campaign.Status,
			StartAt:                  campaign.StartAt,
			EndAt:                    campaign.EndAt,
			TargetMemberCount:        targetMemberCount,
			PaidOrderCount:           paidOrderCount,
			ConvertedMemberCount:     convertedMemberCount,
			RepurchaseConvertedCount: repurchaseConvertedCount,
			RevenueCents:             revenue.RevenueCents,
			ConversionRate:           conversionRate,
		})
	}

	return rows, nil
}

func buildCampaignAttributionCSV(rows []campaignAttributionRow) (string, error) {
	buffer := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buffer)

	header := []string{
		"campaign_id",
		"campaign_name",
		"channel",
		"status",
		"start_at",
		"end_at",
		"target_member_count",
		"paid_order_count",
		"converted_member_count",
		"repurchase_converted_count",
		"revenue_cents",
		"conversion_rate",
	}
	if err := writer.Write(header); err != nil {
		return "", err
	}

	for _, row := range rows {
		record := []string{
			strconv.FormatUint(uint64(row.CampaignID), 10),
			row.CampaignName,
			row.Channel,
			row.Status,
			formatRFC3339(row.StartAt),
			formatRFC3339(row.EndAt),
			strconv.FormatInt(row.TargetMemberCount, 10),
			strconv.FormatInt(row.PaidOrderCount, 10),
			strconv.FormatInt(row.ConvertedMemberCount, 10),
			strconv.FormatInt(row.RepurchaseConvertedCount, 10),
			strconv.FormatInt(row.RevenueCents, 10),
			strconv.FormatFloat(row.ConversionRate, 'f', 2, 64),
		}
		if err := writer.Write(record); err != nil {
			return "", err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func formatRFC3339(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
