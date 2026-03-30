// Package menu 菜单模块
// 提供菜单管理功能：树形菜单结构、分组管理、CRUD、排序与移动
// 依赖 user 模块（用于获取用户信息）
package menu

import (
	"gocms/internal/core"
	"gocms/internal/module/menu/controller"
	"gocms/internal/module/menu/logic"
	"gocms/internal/module/menu/model"
)

// Module menu 模块实现
type Module struct {
	logic *logic.Logic
}

// New 创建 menu 模块实例
func New() *Module {
	return &Module{}
}

// --- DependencyAware 接口 ---

// Dependencies 声明模块依赖
func (m *Module) Dependencies() []string {
	return []string{"user"}
}

// --- Module 接口（必须实现） ---

func (m *Module) Name() string        { return "menu" }
func (m *Module) Description() string { return "菜单管理" }

// Init 初始化菜单模块
// 1. 数据库迁移（创建 menu_items 表）
// 2. 创建 MenuLogic 并注册为 Service
// 3. 初始化默认菜单（首次启动时）
func (m *Module) Init(app *core.App) error {
	// 1. 数据库迁移
	if err := app.DB.AutoMigrate(&model.MenuItem{}); err != nil {
		return err
	}

	// 2. 创建业务逻辑并注册 Service
	m.logic = logic.NewLogic(app.DB, app.Events)
	app.RegisterService("menu", m.logic)

	// 3. 初始化默认菜单（首次启动时）
	if err := m.logic.InitDefaultMenus(); err != nil {
		app.Logger.Warningf(nil, "[menu] 初始化默认菜单失败: %v", err)
	}

	return nil
}

// RegisterRoutes 注册菜单相关 API 路由
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 公开 API：获取菜单树（/api/menus/{group}）
	rg.Public.Bind(controller.NewPublicController(m.logic))

	// 管理 API：菜单 CRUD（/api/admin/*）
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindContent,
		Permissions: []core.PermissionDef{
			{Action: "create", Description: "创建菜单项"},
			{Action: "read", Description: "查看菜单", Scopes: []string{"all"}},
			{Action: "update", Description: "编辑菜单项"},
			{Action: "delete", Description: "删除菜单项"},
			{Action: "manage", Description: "管理菜单（排序、移动）"},
		},
	}
}
