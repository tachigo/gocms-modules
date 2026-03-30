// Package media 媒体模块
package media

import (
	"os"

	"gocms/internal/core"
	"gocms/internal/module/media/controller"
	"gocms/internal/module/media/logic"
	"gocms/internal/module/media/model"
)

func init() {
	core.Register(&Module{})
}

type Module struct {
	logic *logic.Logic
}

func (m *Module) Name() string        { return "media" }
func (m *Module) Description() string { return "媒体文件管理" }
func (m *Module) Version() string     { return "1.0.0" }
func (m *Module) Dependencies() []string { return []string{"user"} }

func (m *Module) Init(app *core.App) error {
	if err := app.DB.AutoMigrate(&model.Media{}, &model.MediaFolder{}); err != nil {
		return err
	}
	uploadPath := "data/uploads"
	os.MkdirAll(uploadPath, 0755)
	m.logic = logic.NewLogic(app.DB, app.Events, uploadPath)
	app.RegisterService("media", m.logic)
	return nil
}

func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

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
