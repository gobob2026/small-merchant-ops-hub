package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"small-merchant-ops-hub-server/internal/cache"
	"small-merchant-ops-hub-server/internal/config"
	"small-merchant-ops-hub-server/internal/db"
)

type testEnvelope[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type testMember struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type testOrder struct {
	ID          uint  `json:"id"`
	AmountCents int64 `json:"amountCents"`
	MemberID    uint  `json:"memberId"`
}

type testSummary struct {
	MemberCount         int64   `json:"memberCount"`
	OrderCount          int64   `json:"orderCount"`
	PaidOrderCount      int64   `json:"paidOrderCount"`
	RevenueCents        int64   `json:"revenueCents"`
	RepurchaseCount     int64   `json:"repurchaseCount"`
	RepurchaseRate      float64 `json:"repurchaseRate"`
	ActiveCampaignCount int64   `json:"activeCampaignCount"`
}

type testCampaign struct {
	ID     uint   `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

type testFollowup struct {
	MemberID       uint  `json:"memberId"`
	PaidOrderCount int64 `json:"paidOrderCount"`
}

type testFollowupPayload struct {
	DaysWindow int            `json:"daysWindow"`
	Items      []testFollowup `json:"items"`
}

func TestMerchantFlowSmoke(t *testing.T) {
	t.Parallel()

	cfg := config.Config{
		Env:             "local",
		Port:            "8080",
		SQLitePath:      filepath.Join(t.TempDir(), "app.db"),
		CacheMode:       "local",
		CORSAllowOrigin: "*",
	}

	database, err := db.Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	cacheStore, err := cache.New(cfg)
	if err != nil {
		t.Fatalf("open cache: %v", err)
	}
	t.Cleanup(func() {
		_ = cacheStore.Close()
	})

	router := NewRouter(database, cacheStore, cfg)

	member := performJSONRequest[testMember](t, router, http.MethodPost, "/api/v1/members", map[string]interface{}{
		"name":    "Alice",
		"phone":   "13800001111",
		"channel": "wechat",
	})
	if member.Code != 200 {
		t.Fatalf("create member code = %d, msg = %s", member.Code, member.Msg)
	}

	firstOrder := performJSONRequest[testOrder](t, router, http.MethodPost, "/api/v1/orders", map[string]interface{}{
		"memberId":    member.Data.ID,
		"amountCents": int64(1200),
		"status":      "paid",
		"source":      "wechat",
	})
	if firstOrder.Code != 200 {
		t.Fatalf("create first order code = %d, msg = %s", firstOrder.Code, firstOrder.Msg)
	}

	summaryBefore := performJSONRequest[testSummary](t, router, http.MethodGet, "/api/v1/summary", nil)
	if summaryBefore.Data.RepurchaseCount != 0 {
		t.Fatalf("repurchase before second order = %d, want 0", summaryBefore.Data.RepurchaseCount)
	}

	secondOrder := performJSONRequest[testOrder](t, router, http.MethodPost, "/api/v1/orders", map[string]interface{}{
		"memberId":    member.Data.ID,
		"amountCents": int64(3400),
		"status":      "paid",
		"source":      "douyin",
	})
	if secondOrder.Code != 200 {
		t.Fatalf("create second order code = %d, msg = %s", secondOrder.Code, secondOrder.Msg)
	}

	member2 := performJSONRequest[testMember](t, router, http.MethodPost, "/api/v1/members", map[string]interface{}{
		"name":    "Bob",
		"phone":   "13800002222",
		"channel": "douyin",
	})
	if member2.Code != 200 {
		t.Fatalf("create second member code = %d, msg = %s", member2.Code, member2.Msg)
	}

	thirdOrder := performJSONRequest[testOrder](t, router, http.MethodPost, "/api/v1/orders", map[string]interface{}{
		"memberId":    member2.Data.ID,
		"amountCents": int64(9900),
		"status":      "paid",
		"source":      "douyin",
	})
	if thirdOrder.Code != 200 {
		t.Fatalf("create third order code = %d, msg = %s", thirdOrder.Code, thirdOrder.Msg)
	}

	campaign := performJSONRequest[testCampaign](t, router, http.MethodPost, "/api/v1/campaigns", map[string]interface{}{
		"name":        "Spring Repurchase",
		"channel":     "wechat",
		"discountPct": 12.5,
		"status":      "active",
	})
	if campaign.Code != 200 {
		t.Fatalf("create campaign code = %d, msg = %s", campaign.Code, campaign.Msg)
	}

	summaryAfter := performJSONRequest[testSummary](t, router, http.MethodGet, "/api/v1/summary", nil)
	if summaryAfter.Data.MemberCount != 2 {
		t.Fatalf("memberCount = %d, want 2", summaryAfter.Data.MemberCount)
	}
	if summaryAfter.Data.OrderCount != 3 {
		t.Fatalf("orderCount = %d, want 3", summaryAfter.Data.OrderCount)
	}
	if summaryAfter.Data.PaidOrderCount != 3 {
		t.Fatalf("paidOrderCount = %d, want 3", summaryAfter.Data.PaidOrderCount)
	}
	if summaryAfter.Data.RevenueCents != 14500 {
		t.Fatalf("revenueCents = %d, want 14500", summaryAfter.Data.RevenueCents)
	}
	if summaryAfter.Data.RepurchaseCount != 1 {
		t.Fatalf("repurchaseCount = %d, want 1", summaryAfter.Data.RepurchaseCount)
	}
	if summaryAfter.Data.RepurchaseRate != 50 {
		t.Fatalf("repurchaseRate = %.2f, want 50", summaryAfter.Data.RepurchaseRate)
	}
	if summaryAfter.Data.ActiveCampaignCount != 1 {
		t.Fatalf("activeCampaignCount = %d, want 1", summaryAfter.Data.ActiveCampaignCount)
	}

	memberList := performJSONRequest[[]testMember](t, router, http.MethodGet, "/api/v1/members", nil)
	if len(memberList.Data) != 2 {
		t.Fatalf("members length = %d, want 2", len(memberList.Data))
	}

	orderList := performJSONRequest[[]testOrder](t, router, http.MethodGet, "/api/v1/orders", nil)
	if len(orderList.Data) != 3 {
		t.Fatalf("orders length = %d, want 3", len(orderList.Data))
	}

	campaignList := performJSONRequest[[]testCampaign](t, router, http.MethodGet, "/api/v1/campaigns", nil)
	if len(campaignList.Data) != 1 {
		t.Fatalf("campaigns length = %d, want 1", len(campaignList.Data))
	}

	followupList := performJSONRequest[testFollowupPayload](t, router, http.MethodGet, "/api/v1/followups", nil)
	if len(followupList.Data.Items) != 1 {
		t.Fatalf("followups length = %d, want 1", len(followupList.Data.Items))
	}
	if followupList.Data.Items[0].MemberID != member2.Data.ID {
		t.Fatalf("followup memberId = %d, want %d", followupList.Data.Items[0].MemberID, member2.Data.ID)
	}
	if followupList.Data.Items[0].PaidOrderCount != 1 {
		t.Fatalf("followup paidOrderCount = %d, want 1", followupList.Data.Items[0].PaidOrderCount)
	}
}

func performJSONRequest[T any](
	t *testing.T,
	router http.Handler,
	method, target string,
	payload interface{},
) testEnvelope[T] {
	t.Helper()

	var body *bytes.Reader
	if payload == nil {
		body = bytes.NewReader(nil)
	} else {
		raw, err := json.Marshal(payload)
		if err != nil {
			t.Fatalf("marshal payload: %v", err)
		}
		body = bytes.NewReader(raw)
	}

	req := httptest.NewRequest(method, target, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("http status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var result testEnvelope[T]
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}
