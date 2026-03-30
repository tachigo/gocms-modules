// Package permission RBAC 中间件
// 拦截 Admin 路由进行权限检查
package permission

import (
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/internal/module/permission/logic"
)

// ---------------------------------------------------------------------------
// RBAC 中间件
// ---------------------------------------------------------------------------

// RBACMiddleware RBAC 权限检查中间件
// 从 JWT 中提取用户信息，检查是否有权限访问当前资源
func (m *Module) RBACMiddleware(r *ghttp.Request) {
	// 获取当前用户 ID
	userID := r.GetCtxVar("user_id").Int64()
	if userID == 0 {
		r.Response.Status = http.StatusUnauthorized
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "未登录",
		})
		return
	}

	// 解析请求路径，确定目标 Module 和 Action
	module, action := parseRequestPath(r.URL.Path, r.Method)

	// admin 用户直接放行（超管特权）
	if m.permissionLogic.IsAdmin(userID) {
		r.Middleware.Next()
		return
	}

	// 检查权限
	hasPerm, scope, err := m.permissionLogic.CheckPermission(userID, module, action)
	if err != nil {
		r.Response.Status = http.StatusInternalServerError
		r.Response.WriteJsonExit(g.Map{
			"code":    500,
			"message": "权限检查失败",
		})
		return
	}

	if !hasPerm {
		r.Response.Status = http.StatusForbidden
		r.Response.WriteJsonExit(g.Map{
			"code":    403,
			"message": "没有权限执行此操作",
		})
		return
	}

	// 将数据范围和用户ID注入 context（供后续查询过滤数据）
	r.SetCtxVar("rbac_scope", scope)   // "all" 或 "own"
	r.SetCtxVar("rbac_user_id", userID)

	r.Middleware.Next()
}

// ---------------------------------------------------------------------------
// 路径解析
// ---------------------------------------------------------------------------

// routeToModule 路由路径段 → 权限模块名 映射表
// URL 路径使用复数/连字符形式，权限表存储 Module.Name()（单数形式）
var routeToModule = map[string]string{
	"articles":    "article",
	"pages":       "page",
	"users":       "user",
	"menus":       "menu",
	"menu-groups": "menu",
	"taxonomies":  "taxonomy",
	"roles":       "permission",
	"permissions": "permission",
	// media、settings 路径与模块名一致，无需映射
}

// normalizeModule 将 URL 路径段归一化为权限表中的模块名
func normalizeModule(urlSegment string) string {
	if mapped, ok := routeToModule[urlSegment]; ok {
		return mapped
	}
	return urlSegment
}

// parseRequestPath 解析请求路径，返回 (module, action)
// 路径格式: /api/admin/{module}/... 或 /api/admin/{module}/{id}
// 根据 HTTP 方法和路径确定操作
func parseRequestPath(path, method string) (module, action string) {
	// 移除 /api/admin/ 前缀
	prefix := "/api/admin/"
	if !strings.HasPrefix(path, prefix) {
		return "", ""
	}

	rest := path[len(prefix):]
	parts := strings.SplitN(rest, "/", 3)
	if len(parts) == 0 {
		return "", ""
	}

	// 第一个部分是路由路径段，归一化为权限模块名
	module = normalizeModule(parts[0])

	// 根据路径和 HTTP 方法确定操作
	hasIDParam := len(parts) > 1 && parts[1] != ""

	switch method {
	case "GET":
		if hasIDParam {
			action = "read"
		} else {
			action = "read" // 列表也是 read
		}
	case "POST":
		action = "create"
	case "PUT", "PATCH":
		action = "update"
	case "DELETE":
		action = "delete"
	default:
		action = "read"
	}

	return module, action
}

// ---------------------------------------------------------------------------
// 公开接口：权限检查辅助方法
// ---------------------------------------------------------------------------

// PermissionLogic 返回权限逻辑实例（供其他模块使用）
func (m *Module) PermissionLogic() *logic.PermissionLogic {
	return m.permissionLogic
}

// CheckUserPermission 检查指定用户是否有权限（供其他模块调用）
func (m *Module) CheckUserPermission(userID int64, module, action string) (bool, string, error) {
	return m.permissionLogic.CheckPermission(userID, module, action)
}

// GetUserRoles 获取用户的角色列表（供其他模块调用）
func (m *Module) GetUserRoles(userID int64) ([]string, error) {
	roles, err := m.permissionLogic.GetUserRoles(userID)
	if err != nil {
		return nil, err
	}

	names := make([]string, 0, len(roles))
	for _, r := range roles {
		names = append(names, r.Name)
	}
	return names, nil
}

// IsAdmin 检查用户是否是管理员（供其他模块调用）
func (m *Module) IsAdmin(userID int64) bool {
	return m.permissionLogic.IsAdmin(userID)
}
