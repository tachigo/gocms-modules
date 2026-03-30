// Package user 用户模块
// 提供用户认证（JWT）、个人信息管理、用户 CRUD
package user

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

func (m *Module) Name() string        { return "user" }
func (m *Module) Description() string { return "用户认证与管理" }
func (m *Module) Version() string     { return "1.0.0" }
func (m *Module) Dependencies() []string { return []string{} }

func (m *Module) Init(app *core.App) error {
	if err := app.DB.AutoMigrate(&model.User{}); err != nil {
		return err
	}
	m.jwtMgr = logic.NewJWTManager(app.Config.JWT.Secret, app.Config.JWT.Expire, app.Config.JWT.Issuer)
	m.userLogic = logic.NewUserLogic(app.DB, m.jwtMgr, app.Events)
	app.RegisterService("user", m.userLogic)
	app.AddAuthMiddleware(m.JWTMiddleware)
	if err := m.userLogic.InitAdmin(); err != nil {
		app.Logger.Warningf(nil, "[user] 初始化管理员失败: %v", err)
	}
	return nil
}

func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	rg.Public.Bind(controller.NewAuthPublicController(m.userLogic))
	rg.Authenticated.Bind(controller.NewAuthProtectedController(m.userLogic))
	rg.Admin.Bind(controller.NewUserAdminController(m.userLogic))
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
	r.SetCtxVar("user_id", claims.UserID)
	r.SetCtxVar("username", claims.Username)
	r.Middleware.Next()
}
