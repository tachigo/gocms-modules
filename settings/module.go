// Package settings 站点配置模块
package settings

import (
	"gocms/internal/core"
	"gocms/internal/module/settings/controller"
	"gocms/internal/module/settings/logic"
)

func init() {
	core.Register(&Module{})
}

type Module struct {
	logic *logic.Logic
}

func (m *Module) Name() string        { return "settings" }
func (m *Module) Description() string { return "站点配置管理" }
func (m *Module) Version() string     { return "1.0.0" }
func (m *Module) Dependencies() []string { return []string{} }

func (m *Module) Init(app *core.App) error {
	m.logic = logic.NewLogic()
	if err := m.logic.LoadFromFile("config/site.yaml"); err != nil {
		app.Logger.Warningf(nil, "[settings] 加载 site.yaml 失败: %v", err)
	}
	app.RegisterService("settings", m.logic)
	return nil
}

func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	rg.Public.Bind(controller.NewPublicController(m.logic))
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindInfrastructure,
		Permissions: []core.PermissionDef{
			{Action: "manage", Description: "管理站点配置"},
		},
	}
}
