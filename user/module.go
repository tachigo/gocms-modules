// Package user 用户模块
// 提供用户认证（JWT）、个人信息管理、用户 CRUD
// 同时向框架注册 JWT 认证中间件，供 Authenticated 和 Admin 路由组使用
package user
import "gocms/internal/core"

func init() {
	core.Register(&Module{})
}


import (
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/internal/core"
	"gocms/internal/module/user/controller"
	"gocms/internal/module/user/logic"
	"gocms/internal/module/user/model"
)

// init 自注册到全局注册表
func init() {
	core.Register(&Module{})
}

// Module user 模块实现
type Module struct {
	userLogic *logic.UserLogic
	jwtMgr    *logic.JWTManager
}

// --- Module 接口（必须实现） ---

func (m *Module) Name() string        { return "user" }
func (m *Module) Description() string { return "用户认证与管理" }
func (m *Module) Version() string     { return "1.0.0" }

// Dependencies 声明依赖
func (m *Module) Dependencies() []string { return []string{} }

// Init 初始化用户模块
// 1. 数据库迁移（创建 users 表）
// 2. 创建 JWT Manager
// 3. 创建 UserLogic 并注册为 Service
// 4. 注册 JWT 认证中间件到框架
// 5. 初始化默认管理员
func (m *Module) Init(app *core.App) error {
	// 1. 数据库迁移
	if err := app.DB.AutoMigrate(&model.User{}); err != nil {
		return err
	}

	// 2. 创建 JWT Manager
	m.jwtMgr = logic.NewJWTManager(
		app.Config.JWT.Secret,
		app.Config.JWT.Expire,
		app.Config.JWT.Issuer,
	)

	// 3. 创建业务逻辑并注册 Service
	m.userLogic = logic.NewUserLogic(app.DB, m.jwtMgr, app.Events)
	app.RegisterService("user", m.userLogic)

	// 4. 注册 JWT 认证中间件（对 Authenticated + Admin 路由组生效）
	app.AddAuthMiddleware(m.JWTMiddleware)

	// 5. 初始化默认管理员（首次启动时自动创建 admin/admin123）
	if err := m.userLogic.InitAdmin(); err != nil {
		app.Logger.Warningf(nil, "[user] 初始化管理员失败: %v", err)
	}

	return nil
}

// RegisterRoutes 注册用户相关 API 路由
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 公开 API：登录（无需认证）
	rg.Public.Bind(controller.NewAuthPublicController(m.userLogic))

	// 需认证 API：登出/个人信息/改密码（JWT 中间件已由框架注入）
	rg.Authenticated.Bind(controller.NewAuthProtectedController(m.userLogic))

	// 管理 API：用户 CRUD（JWT + RBAC 中间件已由框架注入）
	rg.Admin.Bind(controller.NewUserAdminController(m.userLogic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindInfrastructure,
		Permissions: []core.PermissionDef{
			{Action: "create", Description: "创建用户"},
			{Action: "read", Description: "查看用户", Scopes: []string{"own", "all"}},
			{Action: "update", Description: "编辑用户", Scopes: []string{"own", "all"}},
			{Action: "delete", Description: "删除用户"},
		},
	}
}

// ---------------------------------------------------------------------------
// JWT 认证中间件
// ---------------------------------------------------------------------------

// JWTMiddleware JWT 认证中间件
// 从 Authorization header 提取并验证 Token，将用户信息注入 context
func (m *Module) JWTMiddleware(r *ghttp.Request) {
	// 提取 Token
	auth := r.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		r.Response.Status = http.StatusUnauthorized
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "未提供认证Token",
		})
		return
	}
	tokenString := auth[7:]

	// 验证 Token
	claims, err := m.jwtMgr.ParseToken(tokenString)
	if err != nil {
		r.Response.Status = http.StatusUnauthorized
		r.Response.WriteJsonExit(g.Map{
			"code":    401,
			"message": "认证失败",
		})
		return
	}

	// 将用户信息注入 context（后续 handler 通过 GetCurrentUserID 获取）
	r.SetCtxVar("user_id", claims.UserID)
	r.SetCtxVar("username", claims.Username)
	r.Middleware.Next()
}
