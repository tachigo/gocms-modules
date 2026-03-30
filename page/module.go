// Package page 页面模块
// 提供静态页面管理（关于我们、联系方式、服务条款等）
// 支持草稿/发布状态管理、slug 路由、SEO 元数据
// 依赖 user 模块（作者身份）和 media 模块（特色图片）
package page
import "gocms/internal/core"

func init() {
	core.Register(&Module{})
}


import (
	"gocms/internal/core"
	"gocms/internal/module/page/controller"
	"gocms/internal/module/page/logic"
	"gocms/internal/module/page/model"
)

// Module page 模块实现
type Module struct {
	logic *logic.Logic
}

// New 创建 page 模块实例
// init 自注册
func init() {
	core.Register(&Module{})
}

func New() *Module {
	return &Module{}
}

// --- Module 接口（必须实现） ---

func (m *Module) Name() string        { return "page" }
func (m *Module) Description() string  { return "页面管理" }

// Dependencies 声明依赖 user 和 media 模块
func (m *Module) Dependencies() []string { return []string{"user", "media"} }

// Init 初始化页面模块
func (m *Module) Init(app *core.App) error {
	// 1. 数据库迁移
	if err := app.DB.AutoMigrate(&model.Page{}); err != nil {
		return err
	}

	// 2. 创建业务逻辑并注册 Service
	m.logic = logic.NewLogic(app.DB, app.Events)
	app.RegisterService("page", m.logic)

	return nil
}

// RegisterRoutes 注册页面 API 路由
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 公开 API：已发布页面查询（/api/pages）
	rg.Public.Bind(controller.NewPublicController(m.logic))

	// 管理 API：页面 CRUD + 发布管理（/api/admin/pages）
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindContent,
		Fields: []core.FieldDef{
			{ID: "title", Type: core.FieldText, Label: "标题", Required: true},
			{ID: "slug", Type: core.FieldSlug, Label: "URL Slug", Required: true},
			{ID: "body", Type: core.FieldRichtext, Label: "内容"},
			{ID: "excerpt", Type: core.FieldTextarea, Label: "摘要"},
			{ID: "featured_image", Type: core.FieldImage, Label: "特色图片"},
			{ID: "template", Type: core.FieldText, Label: "页面模板"},
			{ID: "sort_order", Type: core.FieldNumber, Label: "排序权重", Default: 0},
			{ID: "meta", Type: core.FieldJSON, Label: "SEO 元数据"},
		},
		Groups: []core.FieldGroup{
			{ID: "main", Label: "基本信息", Fields: []string{"title", "slug", "body", "excerpt"}},
			{ID: "media", Label: "媒体", Fields: []string{"featured_image"}},
			{ID: "settings", Label: "设置", Fields: []string{"template", "sort_order", "meta"}},
		},
		Permissions: []core.PermissionDef{
			{Action: "create", Description: "创建页面"},
			{Action: "read", Description: "查看页面", Scopes: []string{"own", "all"}},
			{Action: "update", Description: "编辑页面", Scopes: []string{"own", "all"}},
			{Action: "delete", Description: "删除页面"},
			{Action: "publish", Description: "发布/取消发布页面"},
		},
		Relations: []core.RelationDef{
			{Field: "author_id", Target: "user", Type: "belongs_to", OnDelete: "restrict"},
		},
	}
}
