// Package taxonomy 分类管理模块
// 提供分类体系（词汇表 + 术语）管理功能
// 依赖 user 模块（通过 JWT 获取操作用户身份）
package taxonomy

import (
	"gorm.io/gorm"

	"gocms/internal/core"
	"gocms/internal/module/taxonomy/controller"
	"gocms/internal/module/taxonomy/logic"
	"gocms/internal/module/taxonomy/model"
)

// Module taxonomy 模块实现
type Module struct {
	logic *logic.Logic
	db    *gorm.DB
}

// New 创建 taxonomy 模块实例
func New() *Module {
	return &Module{}
}

// --- Module 接口（必须实现） ---

func (m *Module) Name() string        { return "taxonomy" }
func (m *Module) Description() string { return "分类管理（词汇表 + 术语）" }

// Dependencies 声明依赖 user 模块
func (m *Module) Dependencies() []string { return []string{"user"} }

// Init 初始化 taxonomy 模块
// 1. 数据库迁移（创建 vocabularies 和 terms 表）
// 2. 创建业务逻辑并注册 Service
// 3. 初始化默认词汇表（categories 和 tags）
func (m *Module) Init(app *core.App) error {
	// 1. 数据库迁移
	if err := app.DB.AutoMigrate(&model.Vocabulary{}, &model.Term{}); err != nil {
		return err
	}

	// 保存 DB 引用（用于初始化默认数据）
	m.db = app.DB

	// 2. 创建业务逻辑
	m.logic = logic.NewLogic(app.DB, app.Events)
	app.RegisterService("taxonomy", m.logic)

	// 3. 初始化默认词汇表
	if err := m.initDefaultVocabularies(); err != nil {
		app.Logger.Warningf(nil, "[taxonomy] 初始化默认词汇表失败: %v", err)
	}

	return nil
}

// RegisterRoutes 注册 taxonomy API 路由
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {
	// 公开 API：/api/taxonomies/...（无需认证）
	rg.Public.Bind(controller.NewPublicController(m.logic))

	// 管理 API：/api/admin/taxonomies/...（需 JWT + RBAC）
	rg.Admin.Bind(controller.NewAdminController(m.logic))
}

// Schema 模块元信息声明
func (m *Module) Schema() core.ModuleSchema {
	return core.ModuleSchema{
		Kind: core.KindContent,
		Permissions: []core.PermissionDef{
			{Action: "create", Description: "创建术语"},
			{Action: "read", Description: "查看术语", Scopes: []string{"all"}},
			{Action: "update", Description: "编辑术语"},
			{Action: "delete", Description: "删除术语"},
		},
	}
}

// ---------------------------------------------------------------------------
// 初始化默认词汇表
// ---------------------------------------------------------------------------

// initDefaultVocabularies 初始化默认词汇表
func (m *Module) initDefaultVocabularies() error {
	// 检查是否已有词汇表
	var count int64
	if err := m.db.Model(&model.Vocabulary{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有词汇表，跳过
	}

	// 创建默认词汇表：categories（文章分类）
	vocab1 := model.Vocabulary{
		MachineID:   "categories",
		Name:        "文章分类",
		Description: "文章的分类体系",
		Hierarchy:   true,
		Weight:      10,
	}
	if err := m.db.Create(&vocab1).Error; err != nil {
		return err
	}

	// 创建默认词汇表：tags（标签）
	vocab2 := model.Vocabulary{
		MachineID:   "tags",
		Name:        "标签",
		Description: "文章标签",
		Hierarchy:   false,
		Weight:      5,
	}
	if err := m.db.Create(&vocab2).Error; err != nil {
		return err
	}

	return nil
}
