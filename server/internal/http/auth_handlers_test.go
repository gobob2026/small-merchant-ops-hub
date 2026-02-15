package http

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"small-merchant-ops-hub-server/internal/cache"
	"small-merchant-ops-hub-server/internal/config"
	"small-merchant-ops-hub-server/internal/db"
)

type authEnvelope[T any] struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data T      `json:"data"`
}

type authLoginData struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

type authPaginated[T any] struct {
	Records []T `json:"records"`
	Current int `json:"current"`
	Size    int `json:"size"`
	Total   int `json:"total"`
}

func TestAuthRoutesSmoke(t *testing.T) {
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

	unauthorized := performJSONRequestWithHeaders[map[string]interface{}](
		t,
		router,
		http.MethodGet,
		"/api/user/info",
		nil,
		nil,
	)
	if unauthorized.Code != 401 {
		t.Fatalf("unauthorized code = %d, want 401", unauthorized.Code)
	}

	invalidLogin := performJSONRequestWithHeaders[map[string]interface{}](
		t,
		router,
		http.MethodPost,
		"/api/auth/login",
		map[string]string{
			"userName": "Super",
			"password": "wrong-password",
		},
		nil,
	)
	if invalidLogin.Code != 401 {
		t.Fatalf("invalid login code = %d, want 401", invalidLogin.Code)
	}

	superLogin := performJSONRequestWithHeaders[authLoginData](
		t,
		router,
		http.MethodPost,
		"/api/auth/login",
		map[string]string{
			"userName": "Super",
			"password": "123456",
		},
		nil,
	)
	if superLogin.Code != 200 {
		t.Fatalf("super login code = %d, msg = %s", superLogin.Code, superLogin.Msg)
	}
	if superLogin.Data.Token == "" || superLogin.Data.RefreshToken == "" {
		t.Fatalf("super login token or refreshToken is empty")
	}

	superInfo := performJSONRequestWithHeaders[authSession](
		t,
		router,
		http.MethodGet,
		"/api/user/info",
		nil,
		map[string]string{
			"Authorization": superLogin.Data.Token,
		},
	)
	if superInfo.Code != 200 {
		t.Fatalf("super info code = %d, msg = %s", superInfo.Code, superInfo.Msg)
	}
	if superInfo.Data.UserName != "Super" {
		t.Fatalf("super userName = %s, want Super", superInfo.Data.UserName)
	}
	if !containsString(superInfo.Data.Buttons, "campaign:create") {
		t.Fatalf("super buttons missing campaign:create: %+v", superInfo.Data.Buttons)
	}
	if !containsString(superInfo.Data.Buttons, "report:export") {
		t.Fatalf("super buttons missing report:export: %+v", superInfo.Data.Buttons)
	}

	originalTTL := authSessionTTL
	authSessionTTL = -1 * time.Second
	expiredLogin := performJSONRequestWithHeaders[authLoginData](
		t,
		router,
		http.MethodPost,
		"/api/auth/login",
		map[string]string{
			"userName": "Super",
			"password": "123456",
		},
		nil,
	)
	authSessionTTL = originalTTL
	if expiredLogin.Code != 200 {
		t.Fatalf("expired login code = %d, msg = %s", expiredLogin.Code, expiredLogin.Msg)
	}
	expiredInfo := performJSONRequestWithHeaders[map[string]interface{}](
		t,
		router,
		http.MethodGet,
		"/api/user/info",
		nil,
		map[string]string{
			"Authorization": expiredLogin.Data.Token,
		},
	)
	if expiredInfo.Code != 401 {
		t.Fatalf("expired token code = %d, want 401", expiredInfo.Code)
	}

	adminLogin := performJSONRequestWithHeaders[authLoginData](
		t,
		router,
		http.MethodPost,
		"/api/auth/login",
		map[string]string{
			"userName": "Admin",
			"password": "123456",
		},
		nil,
	)
	if adminLogin.Code != 200 {
		t.Fatalf("admin login code = %d, msg = %s", adminLogin.Code, adminLogin.Msg)
	}
	if adminLogin.Data.Token == "" {
		t.Fatalf("admin login token is empty")
	}

	adminInfo := performJSONRequestWithHeaders[authSession](
		t,
		router,
		http.MethodGet,
		"/api/user/info",
		nil,
		map[string]string{
			"Authorization": adminLogin.Data.Token,
		},
	)
	if adminInfo.Code != 200 {
		t.Fatalf("admin info code = %d, msg = %s", adminInfo.Code, adminInfo.Msg)
	}
	if adminInfo.Data.UserName != "Admin" {
		t.Fatalf("admin userName = %s, want Admin", adminInfo.Data.UserName)
	}
	if containsString(adminInfo.Data.Buttons, "report:export") {
		t.Fatalf("admin buttons should not include report:export: %+v", adminInfo.Data.Buttons)
	}
	if !containsString(adminInfo.Data.Buttons, "member:create") {
		t.Fatalf("admin buttons missing member:create: %+v", adminInfo.Data.Buttons)
	}

	users := performJSONRequestWithHeaders[authPaginated[userListItem]](
		t,
		router,
		http.MethodGet,
		"/api/user/list?current=1&size=2",
		nil,
		map[string]string{
			"Authorization": superLogin.Data.Token,
		},
	)
	if users.Code != 200 {
		t.Fatalf("user list code = %d, msg = %s", users.Code, users.Msg)
	}
	if users.Data.Total != 3 {
		t.Fatalf("user list total = %d, want 3", users.Data.Total)
	}
	if len(users.Data.Records) != 2 {
		t.Fatalf("user list records length = %d, want 2", len(users.Data.Records))
	}

	roles := performJSONRequestWithHeaders[authPaginated[roleListItem]](
		t,
		router,
		http.MethodGet,
		"/api/role/list?current=1&size=2",
		nil,
		map[string]string{
			"Authorization": superLogin.Data.Token,
		},
	)
	if roles.Code != 200 {
		t.Fatalf("role list code = %d, msg = %s", roles.Code, roles.Msg)
	}
	if roles.Data.Total != 3 {
		t.Fatalf("role list total = %d, want 3", roles.Data.Total)
	}
	if len(roles.Data.Records) != 2 {
		t.Fatalf("role list records length = %d, want 2", len(roles.Data.Records))
	}
}

func performJSONRequestWithHeaders[T any](
	t *testing.T,
	router http.Handler,
	method, target string,
	payload interface{},
	headers map[string]string,
) authEnvelope[T] {
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
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("http status = %d, want 200; body = %s", rec.Code, rec.Body.String())
	}

	var result authEnvelope[T]
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return result
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
