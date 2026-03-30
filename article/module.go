// Package article 文章模块
// 提供文章 CRUD、发布状态管理、分类/标签关联
// 依赖：user, taxonomy, media
package article

import (
	"gocms/internal/core"
	"gocms/internal/module/article/controller"
	"gocms/internal/module/article/logic"
	"gocms/internal/module/article/model"
)

// Module article 模块实现
type Module struct {
	logic *logic.Logic
}

// New 创建 article 模块实例
func New() *Module {
	return &Module{}
}

// --- Module 接口（必须实现） ---

// Name 返回模块唯一标识符
func (m *Module) Name() string { return "article" }

// Description 返回模块描述
func (m *Module) Description() string { return "文章管理" }

// Dependencies 声明模块依赖
func (m *Module) Dependencies() []string { return []string{"user", "taxonomy", "media"} }

// Init 初始化文章模块
// 1. 数据库迁移（创建 articles 和 article_taxonomies 表）
// 2. 创建 Logic 并注册为 Service
func (m *Module) Init(app *core.App) error {
	// 1. 数据库迁移
	if err := app.DB.AutoMigrate(&model.Article{}, &model.ArticleTaxonomy{}); err != nil {
		return err
	}

	// 2. 创建业务逻辑并注册 Service
	m.logic = logic.NewLogic(app.DB, app.Events)
	app.RegisterService("article", m.logic)

	return nil
}

// RegisterRoutes 注册文章相关 API 路由
// 公开 API（/api/articles）+ 管理 API（/api/admin/articles）
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 公开 API：获取已发布文章（无需认证）
	rg.Public.Bind(controller.NewPublicController(m.logic))

	// 管理 API：文章 CRUD + 发布管理（需 JWT + RBAC）
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindContent,
		Fields: []core.FieldDef{
			{ID: "title", Type: core.FieldText, Label: "标题", Required: true, Validations: map[string]interface{}{"max": 200}},
			{ID: "slug", Type: core.FieldSlug, Label: "URL别名", Required: true, Validations: map[string]interface{}{"max": 200}},
			{ID: "summary", Type: core.FieldTextarea, Label: "摘要"},
			{ID: "body", Type: core.FieldRichtext, Label: "正文", Required: true},
			{ID: "cover_image", Type: core.FieldImage, Label: "封面图"},
			{ID: "status", Type: core.FieldSelect, Label: "状态", Default: "draft", Options: map[string]interface{}{
				"options": []map[string]string{
					{"value": "draft", "label": "草稿"},
					{"value": "published", "label": "已发布"},
					{"value": "archived", "label": "已归档"},
				},
			}},
			{ID: "is_top", Type: core.FieldBoolean, Label: "置顶", Default: false},
			{ID: "seo_title", Type: core.FieldText, Label: "SEO标题"},
			{ID: "seo_desc", Type: core.FieldTextarea, Label: "SEO描述"},
			{ID: "category_ids", Type: core.FieldJSON, Label: "分类"},
			{ID: "tag_ids", Type: core.FieldJSON, Label: "标签"},
		},
		Permissions: []core.PermissionDef{
			{Action: "create", Description: "创建文章"},
			{Action: "read", Description: "查看文章", Scopes: []string{"own", "all"}},
			{Action: "update", Description: "编辑文章", Scopes: []string{"own", "all"}},
			{Action: "delete", Description: "删除文章"},
		},
		Relations: []core.RelationDef{
			{Field: "AuthorID", Target: "user", Type: "belongs_to", OnDelete: "set_null"},
			{Field: "CoverImage", Target: "media", Type: "belongs_to", OnDelete: "set_null"},
			{Field: "CategoryIDs", Target: "taxonomy", Type: "many_to_many"},
			{Field: "TagIDs", Target: "taxonomy", Type: "many_to_many"},
		},
	}
}
