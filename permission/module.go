// Package permission 权限模块
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

type Module struct {
	registry *core.Registry
	logic    *logic.PermissionLogic
}

func (m *Module) Name() string        { return "permission" }
func (m *Module) Description() string { return "RBAC 角色权限管理" }
func (m *Module) Version() string     { return "1.0.0" }
func (m *Module) Dependencies() []string { return []string{"user"} }

func (m *Module) Init(app *core.App) error {
	if err := app.DB.AutoMigrate(&model.Role{}, &model.Permission{}, &model.UserRole{}); err != nil {
		return err
	}
	m.registry = core.GetGlobalRegistry()
	schemas := m.registry.AllSchemas()
	m.logic = logic.NewPermissionLogic(app.DB, app.Events, schemas)
	app.RegisterService("permission", m.logic)
	app.AddAdminMiddleware(m.RBACMiddleware)
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
