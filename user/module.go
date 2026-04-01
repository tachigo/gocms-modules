// Package user 用户模块
// 提供用户认证（JWT）、个人信息管理、用户 CRUD
// 支持两种运行模式：
//   - master: 独立运行，拥有完整的登录/注册/数据库管理功能
//   - slave:  作为SSO客户端运行，依赖上游鉴权，不维护本地用户表
package user

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/core"
	"gocms/module/user/controller"
	"gocms/module/user/logic"
	"gocms/module/user/model"
)

// init 自注册到全局注册表
func init() {
	core.Register(&Module{})
}

// Module user 模块实现
type Module struct {
	userLogic *logic.UserLogic
	jwtMgr    *logic.JWTManager
	mode      string // 运行模式: master | slave
}

func (m *Module) Name() string           { return "user" }
func (m *Module) Description() string    { return "用户认证与管理" }
func (m *Module) Version() string        { return "1.0.0" }
func (m *Module) Dependencies() []string { return []string{} }

// Init 模块初始化
// 根据配置 mode 决定初始化行为：
//   - master: 完整初始化（数据库迁移、管理员创建、JWT中间件）
//   - slave:  简化初始化（仅SSO中间件，无本地数据库操作）
func (m *Module) Init(app *core.App) error {
	// 从配置中读取运行模式，默认为 master
	m.mode = app.Config.User.Mode
	if m.mode == "" {
		m.mode = "master"
	}

	app.Logger.Infof(nil, "[user] 模块运行模式: %s", m.mode)

	// master 模式：执行数据库迁移和初始化
	if m.mode == "master" {
		if err := app.DB.AutoMigrate(&model.User{}); err != nil {
			return err
		}
	}

	// 初始化 JWT 管理器（两种模式都需要，用于验证本地或SSO颁发的Token）
	m.jwtMgr = logic.NewJWTManager(app.Config.JWT.Secret, app.Config.JWT.Expire, app.Config.JWT.Issuer, app.Cache)

	// 创建 UserLogic，传入运行模式
	m.userLogic = logic.NewUserLogic(app.DB, m.jwtMgr, app.Events, m.mode)

	// 注册用户服务
	app.RegisterService("user", m.userLogic)

	// 根据模式添加不同的中间件
	if m.mode == "slave" {
		// slave 模式：添加 SSO 中间件，从请求头解析SSO用户信息
		app.AddAuthMiddleware(m.SSOMiddleware)
	} else {
		// master 模式：添加 JWT 中间件
		app.AddAuthMiddleware(m.JWTMiddleware)
	}

	// master 模式：初始化管理员账号
	if m.mode == "master" {
		if err := m.userLogic.InitAdmin(); err != nil {
			app.Logger.Warningf(nil, "[user] 初始化管理员失败: %v", err)
		}
	}

	return nil
}

// RegisterRoutes 注册路由
// master 模式：注册完整路由（包含登录/注册）
// slave 模式：仅注册用户信息查询路由（登录/注册由SSO系统处理）
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// master 模式：注册公开认证路由（登录/注册）
	if m.mode == "master" {
		rg.Public.Bind(controller.NewAuthPublicController(m.userLogic))
	}

	// 两种模式都注册受保护的用户信息路由
	rg.Authenticated.Bind(controller.NewAuthProtectedController(m.userLogic))

	// master 模式：注册管理员路由
	if m.mode == "master" {
		rg.Admin.Bind(controller.NewUserAdminController(m.userLogic))
	}
}

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

// JWTMiddleware JWT 认证中间件（master 模式使用）
// 从 Authorization 头解析 JWT Token，提取用户信息注入上下文
func (m *Module) JWTMiddleware(r *ghttp.Request) {
	auth := r.GetHeader("Authorization")
	if !strings.HasPrefix(auth, "Bearer ") {
		r.Response.Status = http.StatusUnauthorized
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": "未提供认证Token"})
		return
	}
	claims, err := m.jwtMgr.ParseToken(auth[7:])
	if err != nil {
		r.Response.Status = http.StatusUnauthorized
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": "认证失败"})
		return
	}

	// 将用户信息注入 context（支持 slave 模式的 GetUserFromCtx）
	userInfo := &core.UserInfo{
		ID:       claims.UserID,
		Username: claims.Username,
	}
	r.SetCtx(core.SetUserToCtx(r.GetCtx(), userInfo))
	r.SetCtxVar("user_id", claims.UserID)
	r.Middleware.Next()
}

// SSOMiddleware SSO 认证中间件（slave 模式使用）
// 从请求头解析SSO系统传递的用户信息，注入上下文
// 支持两种SSO集成方式：
//  1. JWT Token 方式：Authorization 头携带SSO颁发的JWT
//  2. Header 透传方式：X-SSO-User-ID 等头直接传递用户信息
func (m *Module) SSOMiddleware(r *ghttp.Request) {
	auth := r.GetHeader("Authorization")

	// 方式1：尝试解析 JWT Token（SSO颁发的令牌）
	if strings.HasPrefix(auth, "Bearer ") {
		claims, err := m.jwtMgr.ParseToken(auth[7:])
		if err == nil {
			// Token 有效，注入用户信息
			userInfo := &core.UserInfo{
				ID:       claims.UserID,
				Username: claims.Username,
			}
			r.SetCtx(core.SetUserToCtx(r.GetCtx(), userInfo))
			r.Middleware.Next()
			return
		}
	}

	// 方式2：从 Header 中直接获取SSO透传的用户信息
	userID := r.GetHeader("X-SSO-User-ID")
	username := r.GetHeader("X-SSO-Username")

	if userID == "" || username == "" {
		r.Response.Status = http.StatusUnauthorized
		r.Response.WriteJsonExit(g.Map{"code": 401, "message": "SSO认证信息缺失"})
		return
	}

	// 解析用户ID
	var uid int64
	fmt.Sscanf(userID, "%d", &uid)

	userInfo := &core.UserInfo{
		ID:       uid,
		Username: username,
		Email:    r.GetHeader("X-SSO-Email"),
		Role:     r.GetHeader("X-SSO-Role"),
	}

	r.SetCtx(core.SetUserToCtx(r.GetCtx(), userInfo))
	r.SetCtxVar("user_id", uid)
	r.Middleware.Next()
}
