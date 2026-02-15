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
	MemberCount     int64   `json:"memberCount"`
	OrderCount      int64   `json:"orderCount"`
	PaidOrderCount  int64   `json:"paidOrderCount"`
	RevenueCents    int64   `json:"revenueCents"`
	RepurchaseCount int64   `json:"repurchaseCount"`
	RepurchaseRate  float64 `json:"repurchaseRate"`
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

	summaryAfter := performJSONRequest[testSummary](t, router, http.MethodGet, "/api/v1/summary", nil)
	if summaryAfter.Data.MemberCount != 1 {
		t.Fatalf("memberCount = %d, want 1", summaryAfter.Data.MemberCount)
	}
	if summaryAfter.Data.OrderCount != 2 {
		t.Fatalf("orderCount = %d, want 2", summaryAfter.Data.OrderCount)
	}
	if summaryAfter.Data.PaidOrderCount != 2 {
		t.Fatalf("paidOrderCount = %d, want 2", summaryAfter.Data.PaidOrderCount)
	}
	if summaryAfter.Data.RevenueCents != 4600 {
		t.Fatalf("revenueCents = %d, want 4600", summaryAfter.Data.RevenueCents)
	}
	if summaryAfter.Data.RepurchaseCount != 1 {
		t.Fatalf("repurchaseCount = %d, want 1", summaryAfter.Data.RepurchaseCount)
	}
	if summaryAfter.Data.RepurchaseRate != 100 {
		t.Fatalf("repurchaseRate = %.2f, want 100", summaryAfter.Data.RepurchaseRate)
	}

	memberList := performJSONRequest[[]testMember](t, router, http.MethodGet, "/api/v1/members", nil)
	if len(memberList.Data) != 1 {
		t.Fatalf("members length = %d, want 1", len(memberList.Data))
	}

	orderList := performJSONRequest[[]testOrder](t, router, http.MethodGet, "/api/v1/orders", nil)
	if len(orderList.Data) != 2 {
		t.Fatalf("orders length = %d, want 2", len(orderList.Data))
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
