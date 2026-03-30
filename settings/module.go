// Package settings 站点配置模块
// 从 config/site.yaml 加载站点配置，提供公开和管理两套 API
package settings

import (
	"gocms/internal/core"
	"gocms/internal/module/settings/controller"
	"gocms/internal/module/settings/logic"
)

// Module settings 模块实现
type Module struct {
	logic *logic.Logic
}

// New 创建 settings 模块实例
func New() *Module {
	return &Module{}
}

// --- Module 接口（必须实现） ---

func (m *Module) Name() string        { return "settings" }
func (m *Module) Description() string  { return "站点配置管理" }

// Init 初始化：从 config/site.yaml 加载站点配置
func (m *Module) Init(app *core.App) error {
	m.logic = logic.NewLogic()

	// 尝试加载站点配置（文件不存在时使用默认值，不阻塞启动）
	if err := m.logic.LoadFromFile("config/site.yaml"); err != nil {
		app.Logger.Warningf(nil, "[settings] 加载 site.yaml 失败，使用默认配置: %v", err)
	}

	// 注册 Service 供其他模块使用（如 media 获取图片样式配置）
	app.RegisterService("settings", m.logic)
	return nil
}

// RegisterRoutes 注册 API 路由
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 公开 API：GET /api/settings
	rg.Public.Bind(controller.NewPublicController(m.logic))

	// 管理 API：GET /api/admin/settings
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindInfrastructure,
		Permissions: []core.PermissionDef{
			{Action: "manage", Description: "管理站点配置"},
		},
	}
}
