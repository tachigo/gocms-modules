// Package media 媒体模块
// 文件上传/存储/文件夹管理
// 依赖 user 模块（通过 JWT 获取上传者身份）
package media
import "gocms/internal/core"

func init() {
	core.Register(&Module{})
}


import (
	"os"

	"gocms/internal/core"
	"gocms/internal/module/media/controller"
	"gocms/internal/module/media/logic"
	"gocms/internal/module/media/model"
)

// Module media 模块实现
type Module struct {
	logic *logic.Logic
}

// New 创建 media 模块实例
// init 自注册到全局注册表
func init() {
	core.Register(&Module{})
}

// New 创建 media 模块实例
func New() *Module {
	return &Module{}
}

// --- Module 接口 ---

func (m *Module) Name() string        { return "media" }
func (m *Module) Description() string  { return "媒体文件管理" }

// Dependencies 声明依赖 user 模块
func (m *Module) Dependencies() []string { return []string{"user"} }

// Init 初始化媒体模块
func (m *Module) Init(app *core.App) error {
	// 1. 数据库迁移
	if err := app.DB.AutoMigrate(&model.Media{}, &model.MediaFolder{}); err != nil {
		return err
	}

	// 2. 确保上传目录存在
	uploadPath := "data/uploads"
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		return err
	}

	// 3. 创建业务逻辑
	m.logic = logic.NewLogic(app.DB, app.Events, uploadPath)
	app.RegisterService("media", m.logic)

	return nil
}

// RegisterRoutes 注册媒体管理 API
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 管理 API：媒体 CRUD + 文件夹管理
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindInfrastructure,
		Permissions: []core.PermissionDef{
			{Action: "create", Description: "上传文件"},
			{Action: "read", Description: "查看媒体"},
			{Action: "update", Description: "编辑媒体信息"},
			{Action: "delete", Description: "删除文件"},
		},
	}
}
