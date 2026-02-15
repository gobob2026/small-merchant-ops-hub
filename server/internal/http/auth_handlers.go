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

type authMarkItem struct {
	Title    string `json:"title"`
	AuthMark string `json:"authMark"`
}

type menuMeta struct {
	Title     string         `json:"title"`
	Icon      string         `json:"icon,omitempty"`
	KeepAlive bool           `json:"keepAlive,omitempty"`
	IsHide    bool           `json:"isHide,omitempty"`
	IsHideTab bool           `json:"isHideTab,omitempty"`
	Roles     []string       `json:"roles,omitempty"`
	AuthList  []authMarkItem `json:"authList,omitempty"`
}

type menuRoute struct {
	Path      string      `json:"path"`
	Name      string      `json:"name,omitempty"`
	Component string      `json:"component,omitempty"`
	Meta      menuMeta    `json:"meta"`
	Children  []menuRoute `json:"children,omitempty"`
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
	router.GET("/api/v3/system/menus", systemMenuListHandler)
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

func systemMenuListHandler(c *gin.Context) {
	session, found := currentSession(c)
	if !found {
		fail(c, 401, "unauthorized")
		return
	}

	menus := filterMenuRoutesByRoles(baseSystemMenus(), session.Roles)
	ok(c, menus)
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

func baseSystemMenus() []menuRoute {
	return []menuRoute{
		{
			Path:      "/dashboard",
			Name:      "Dashboard",
			Component: "/index/index",
			Meta: menuMeta{
				Title: "仪表盘",
				Icon:  "ri:dashboard-3-line",
				Roles: []string{"R_SUPER", "R_ADMIN"},
			},
			Children: []menuRoute{
				{
					Path:      "analysis",
					Name:      "Analysis",
					Component: "/dashboard/analysis",
					Meta: menuMeta{
						Title:     "分析页",
						Icon:      "ri:line-chart-line",
						KeepAlive: true,
						Roles:     []string{"R_SUPER", "R_ADMIN"},
					},
				},
			},
		},
		{
			Path:      "/operations",
			Name:      "Operations",
			Component: "/index/index",
			Meta: menuMeta{
				Title: "商家运营",
				Icon:  "ri:store-2-line",
				Roles: []string{"R_SUPER", "R_ADMIN", "R_USER"},
			},
			Children: []menuRoute{
				{
					Path:      "hub",
					Name:      "MerchantOpsHub",
					Component: "/operations/hub",
					Meta: menuMeta{
						Title:     "运营台",
						Icon:      "ri:line-chart-line",
						KeepAlive: false,
						Roles:     []string{"R_SUPER", "R_ADMIN", "R_USER"},
						AuthList: []authMarkItem{
							{Title: "新增会员", AuthMark: "member:create"},
							{Title: "新增订单", AuthMark: "order:create"},
							{Title: "新增活动", AuthMark: "campaign:create"},
							{Title: "查看跟进名单", AuthMark: "followup:view"},
							{Title: "导出归因报表", AuthMark: "report:export"},
						},
					},
				},
			},
		},
		{
			Path:      "/system",
			Name:      "System",
			Component: "/index/index",
			Meta: menuMeta{
				Title: "系统管理",
				Icon:  "ri:user-3-line",
				Roles: []string{"R_SUPER", "R_ADMIN"},
			},
			Children: []menuRoute{
				{
					Path:      "user",
					Name:      "User",
					Component: "/system/user",
					Meta: menuMeta{
						Title:     "用户管理",
						Icon:      "ri:user-line",
						KeepAlive: true,
						Roles:     []string{"R_SUPER", "R_ADMIN"},
					},
				},
				{
					Path:      "role",
					Name:      "Role",
					Component: "/system/role",
					Meta: menuMeta{
						Title:     "角色管理",
						Icon:      "ri:user-settings-line",
						KeepAlive: true,
						Roles:     []string{"R_SUPER"},
					},
				},
				{
					Path:      "user-center",
					Name:      "UserCenter",
					Component: "/system/user-center",
					Meta: menuMeta{
						Title:     "个人中心",
						Icon:      "ri:user-line",
						IsHide:    true,
						IsHideTab: true,
						KeepAlive: true,
						Roles:     []string{"R_SUPER", "R_ADMIN"},
					},
				},
				{
					Path:      "menu",
					Name:      "Menus",
					Component: "/system/menu",
					Meta: menuMeta{
						Title:     "菜单管理",
						Icon:      "ri:menu-line",
						KeepAlive: true,
						Roles:     []string{"R_SUPER"},
						AuthList: []authMarkItem{
							{Title: "新增", AuthMark: "add"},
							{Title: "编辑", AuthMark: "edit"},
							{Title: "删除", AuthMark: "delete"},
						},
					},
				},
			},
		},
	}
}

func filterMenuRoutesByRoles(routes []menuRoute, roles []string) []menuRoute {
	filtered := make([]menuRoute, 0, len(routes))
	for _, route := range routes {
		if !hasRoleAccess(route.Meta.Roles, roles) {
			continue
		}

		current := route
		if len(route.Children) > 0 {
			current.Children = filterMenuRoutesByRoles(route.Children, roles)
			if len(current.Children) == 0 {
				continue
			}
		}
		filtered = append(filtered, current)
	}
	return filtered
}

func hasRoleAccess(requiredRoles []string, userRoles []string) bool {
	if len(requiredRoles) == 0 {
		return true
	}
	for _, role := range requiredRoles {
		for _, userRole := range userRoles {
			if role == userRole {
				return true
			}
		}
	}
	return false
}
