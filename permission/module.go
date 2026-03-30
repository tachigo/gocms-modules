// Package permission 权限模块
// 提供 RBAC 角色权限管理：角色 CRUD、权限检查、用户角色分配
// 向框架注册 RBAC 中间件，供 Admin 路由组使用
package permission
import "gocms/internal/core"

func init() {
	core.Register(&Module{})
}


import (
	"gocms/internal/core"
	"gocms/internal/module/permission/controller"
	"gocms/internal/module/permission/logic"
	"gocms/internal/module/permission/model"
)

// init 自注册到全局注册表
func init() {
	core.Register(&Module{})
}

// Module permission 模块实现
type Module struct {
	registry        *core.Registry
	permissionLogic *logic.PermissionLogic
}

// --- Module 接口（必须实现） ---

func (m *Module) Name() string        { return "permission" }
func (m *Module) Description() string { return "RBAC 角色权限管理" }
func (m *Module) Version() string     { return "1.0.0" }

// Dependencies 声明模块依赖
func (m *Module) Dependencies() []string {
	return []string{"user"}
}

// Init 初始化权限模块
// 1. 数据库迁移（创建 roles/permissions/user_roles 表）
// 2. 获取所有 Module Schema 构建权限矩阵
// 3. 创建 PermissionLogic 并注册为 Service
// 4. 注册 RBAC 中间件到框架
// 5. 初始化 3 个默认角色（admin/editor/viewer）
func (m *Module) Init(app *core.App) error {
	// 1. 数据库迁移
	if err := app.DB.AutoMigrate(&model.Role{}, &model.Permission{}, &model.UserRole{}); err != nil {
		return err
	}

	// 2. 获取所有 Module 的 Schema 声明
	schemas := m.registry.AllSchemas()

	// 3. 创建业务逻辑并注册 Service
	m.permissionLogic = logic.NewPermissionLogic(app.DB, app.Events, schemas)
	app.RegisterService("permission", m.permissionLogic)

	// 4. 注册 RBAC 中间件（对 Admin 路由组生效）
	app.AddAdminMiddleware(m.RBACMiddleware)

	// 5. 初始化默认角色（首次启动时自动创建 admin/editor/viewer）
	if err := m.permissionLogic.InitDefaultRoles(); err != nil {
		app.Logger.Warningf(nil, "[permission] 初始化默认角色失败: %v", err)
	}

	return nil
}

// RegisterRoutes 注册权限相关 API 路由
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 管理 API：角色 CRUD + 权限管理（需 JWT + RBAC 中间件）
	rg.Admin.Bind(controller.NewPermissionController(m.permissionLogic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindInfrastructure,
		Permissions: []core.PermissionDef{
			{Action: "manage", Description: "管理角色权限"},
			{Action: "create", Description: "创建角色"},
			{Action: "read", Description: "查看角色"},
			{Action: "update", Description: "编辑角色"},
			{Action: "delete", Description: "删除角色"},
		},
	}
}
