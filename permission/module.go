// Package permission 权限模块
// 提供 RBAC 角色权限管理，支持本地角色和 SSO 角色映射
package permission

import (
	"gocms/internal/core"
	"gocms/internal/module/permission/controller"
	"gocms/internal/module/permission/logic"
	"gocms/internal/module/permission/model"
)

func init() {
	core.Register(&Module{})
}

// Module 权限模块实现
// 支持两种权限来源的并集策略：
//   1. 本地数据库中的 user_roles 关联
//   2. SSO 角色通过 role_mapping 映射到本地角色
type Module struct {
	registry    *core.Registry
	logic       *logic.PermissionLogic
	// roleMapping SSO 角色到本地角色的映射表
	// key: SSO 角色名 (如 "sso_manager")
	// value: 本地角色名 (如 "admin")
	roleMapping map[string]string
}

func (m *Module) Name() string        { return "permission" }
func (m *Module) Description() string { return "RBAC 角色权限管理" }
func (m *Module) Version() string     { return "1.0.0" }
func (m *Module) Dependencies() []string { return []string{"user"} }

// Init 模块初始化
// 从全局 Config 读取 permission.role_mapping 配置
// 初始化数据库表和默认角色
func (m *Module) Init(app *core.App) error {
	// 数据库迁移
	if err := app.DB.AutoMigrate(&model.Role{}, &model.Permission{}, &model.UserRole{}); err != nil {
		return err
	}

	// 从配置读取 SSO 角色映射
	m.roleMapping = make(map[string]string)
	if app.Config.Permission.RoleMapping != nil {
		m.roleMapping = app.Config.Permission.RoleMapping
	}

	m.registry = core.GetGlobalRegistry()
	schemas := m.registry.AllSchemas()

	// 创建 PermissionLogic，传入角色映射配置
	m.logic = logic.NewPermissionLogic(app.DB, app.Events, schemas, m.roleMapping)

	app.RegisterService("permission", m.logic)
	app.AddAdminMiddleware(m.RBACMiddleware)

	// 初始化默认角色
	if err := m.logic.InitDefaultRoles(); err != nil {
		app.Logger.Warningf(nil, "[permission] 初始化默认角色失败: %v", err)
	}

	return nil
}

func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	rg.Admin.Bind(controller.NewPermissionController(m.logic))
}

func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindInfrastructure,
		Permissions: []core.PermissionDef{
			{Action: "manage", Description: "管理角色权限"},
		},
	}
}
