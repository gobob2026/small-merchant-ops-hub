package http

import (
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type loginRequest struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

type authSession struct {
	UserID   int      `json:"userId"`
	UserName string   `json:"userName"`
	Email    string   `json:"email"`
	Avatar   string   `json:"avatar"`
	Roles    []string `json:"roles"`
	Buttons  []string `json:"buttons"`
}

type userListItem struct {
	ID         int      `json:"id"`
	Avatar     string   `json:"avatar"`
	Status     string   `json:"status"`
	UserName   string   `json:"userName"`
	UserGender string   `json:"userGender"`
	NickName   string   `json:"nickName"`
	UserPhone  string   `json:"userPhone"`
	UserEmail  string   `json:"userEmail"`
	UserRoles  []string `json:"userRoles"`
	CreateBy   string   `json:"createBy"`
	CreateTime string   `json:"createTime"`
	UpdateBy   string   `json:"updateBy"`
	UpdateTime string   `json:"updateTime"`
}

type roleListItem struct {
	RoleID      int    `json:"roleId"`
	RoleName    string `json:"roleName"`
	RoleCode    string `json:"roleCode"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
	CreateTime  string `json:"createTime"`
}

type paginatedData struct {
	Records interface{} `json:"records"`
	Current int         `json:"current"`
	Size    int         `json:"size"`
	Total   int         `json:"total"`
}

type authSessionEntry struct {
	Session   authSession
	ExpiresAt time.Time
}

var (
	authSessionTTL = 24 * time.Hour
	authSessions   = map[string]authSessionEntry{}
	authSessionsMu sync.RWMutex
)

func registerAuthRoutes(router *gin.Engine) {
	router.POST("/api/auth/login", loginHandler)
	router.GET("/api/user/info", userInfoHandler)
	router.GET("/api/user/list", userListHandler)
	router.GET("/api/role/list", roleListHandler)
}

func loginHandler(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, "invalid login payload")
		return
	}

	req.UserName = strings.TrimSpace(req.UserName)
	req.Password = strings.TrimSpace(req.Password)
	if req.UserName == "" || req.Password == "" {
		fail(c, 400, "userName and password are required")
		return
	}
	if req.Password != "123456" {
		fail(c, 401, "invalid credentials")
		return
	}

	session, found := resolveSessionByUserName(req.UserName)
	if !found {
		fail(c, 401, "invalid credentials")
		return
	}

	token := fmt.Sprintf("token-%s-%d", strings.ToLower(session.UserName), time.Now().UnixNano())
	refreshToken := fmt.Sprintf("refresh-%d", time.Now().UnixNano())
	saveSession(token, session)

	ok(c, gin.H{
		"token":        token,
		"refreshToken": refreshToken,
	})
}

func userInfoHandler(c *gin.Context) {
	session, found := currentSession(c)
	if !found {
		fail(c, 401, "unauthorized")
		return
	}
	ok(c, session)
}

func userListHandler(c *gin.Context) {
	_, found := currentSession(c)
	if !found {
		fail(c, 401, "unauthorized")
		return
	}

	current := parseIntWithBounds(c.Query("current"), 1, 1, 1000)
	size := parseIntWithBounds(c.Query("size"), 10, 1, 200)

	users := []userListItem{
		{
			ID:         1,
			Avatar:     "",
			Status:     "1",
			UserName:   "Super",
			UserGender: "1",
			NickName:   "Super",
			UserPhone:  "13800001111",
			UserEmail:  "super@merchant.local",
			UserRoles:  []string{"R_SUPER"},
			CreateBy:   "system",
			CreateTime: "2026-02-01 10:00:00",
			UpdateBy:   "system",
			UpdateTime: "2026-02-01 10:00:00",
		},
		{
			ID:         2,
			Avatar:     "",
			Status:     "1",
			UserName:   "Admin",
			UserGender: "1",
			NickName:   "Admin",
			UserPhone:  "13800002222",
			UserEmail:  "admin@merchant.local",
			UserRoles:  []string{"R_ADMIN"},
			CreateBy:   "system",
			CreateTime: "2026-02-01 10:00:00",
			UpdateBy:   "system",
			UpdateTime: "2026-02-01 10:00:00",
		},
		{
			ID:         3,
			Avatar:     "",
			Status:     "1",
			UserName:   "User",
			UserGender: "1",
			NickName:   "User",
			UserPhone:  "13800003333",
			UserEmail:  "user@merchant.local",
			UserRoles:  []string{"R_USER"},
			CreateBy:   "system",
			CreateTime: "2026-02-01 10:00:00",
			UpdateBy:   "system",
			UpdateTime: "2026-02-01 10:00:00",
		},
	}

	paged := paginate(users, current, size)
	ok(c, paginatedData{
		Records: paged,
		Current: current,
		Size:    size,
		Total:   len(users),
	})
}

func roleListHandler(c *gin.Context) {
	_, found := currentSession(c)
	if !found {
		fail(c, 401, "unauthorized")
		return
	}

	current := parseIntWithBounds(c.Query("current"), 1, 1, 1000)
	size := parseIntWithBounds(c.Query("size"), 10, 1, 200)

	roles := []roleListItem{
		{
			RoleID:      1,
			RoleName:    "Super Admin",
			RoleCode:    "R_SUPER",
			Description: "Full access",
			Enabled:     true,
			CreateTime:  "2026-02-01 10:00:00",
		},
		{
			RoleID:      2,
			RoleName:    "Admin",
			RoleCode:    "R_ADMIN",
			Description: "Operations access without export",
			Enabled:     true,
			CreateTime:  "2026-02-01 10:00:00",
		},
		{
			RoleID:      3,
			RoleName:    "User",
			RoleCode:    "R_USER",
			Description: "Read-only business access",
			Enabled:     true,
			CreateTime:  "2026-02-01 10:00:00",
		},
	}

	paged := paginate(roles, current, size)
	ok(c, paginatedData{
		Records: paged,
		Current: current,
		Size:    size,
		Total:   len(roles),
	})
}

func resolveSessionByUserName(userName string) (authSession, bool) {
	switch strings.ToLower(strings.TrimSpace(userName)) {
	case "super":
		return authSession{
			UserID:   1,
			UserName: "Super",
			Email:    "super@merchant.local",
			Avatar:   "",
			Roles:    []string{"R_SUPER"},
			Buttons: []string{
				"member:create",
				"order:create",
				"campaign:create",
				"followup:view",
				"report:export",
			},
		}, true
	case "admin":
		return authSession{
			UserID:   2,
			UserName: "Admin",
			Email:    "admin@merchant.local",
			Avatar:   "",
			Roles:    []string{"R_ADMIN"},
			Buttons: []string{
				"member:create",
				"order:create",
				"followup:view",
			},
		}, true
	case "user":
		return authSession{
			UserID:   3,
			UserName: "User",
			Email:    "user@merchant.local",
			Avatar:   "",
			Roles:    []string{"R_USER"},
			Buttons: []string{
				"followup:view",
			},
		}, true
	default:
		return authSession{}, false
	}
}

func currentSession(c *gin.Context) (authSession, bool) {
	token := parseAuthToken(c.GetHeader("Authorization"))
	if token == "" {
		return authSession{}, false
	}

	authSessionsMu.RLock()
	entry, ok := authSessions[token]
	authSessionsMu.RUnlock()
	if !ok {
		return authSession{}, false
	}
	if time.Now().After(entry.ExpiresAt) {
		removeSession(token)
		return authSession{}, false
	}
	return entry.Session, true
}

func saveSession(token string, session authSession) {
	authSessionsMu.Lock()
	defer authSessionsMu.Unlock()
	authSessions[token] = authSessionEntry{
		Session:   session,
		ExpiresAt: time.Now().Add(authSessionTTL),
	}
}

func removeSession(token string) {
	authSessionsMu.Lock()
	defer authSessionsMu.Unlock()
	delete(authSessions, token)
}

func parseAuthToken(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(raw), "bearer ") {
		return strings.TrimSpace(raw[7:])
	}
	return raw
}

func parseIntWithBounds(raw string, fallback, min, max int) int {
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func paginate[T any](items []T, current, size int) []T {
	if len(items) == 0 {
		return []T{}
	}
	start := (current - 1) * size
	if start >= len(items) {
		return []T{}
	}
	end := start + size
	if end > len(items) {
		end = len(items)
	}
	return items[start:end]
}
