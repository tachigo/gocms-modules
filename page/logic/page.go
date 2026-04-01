// Package logic page 业务逻辑
// 页面 CRUD + 发布/取消发布
package logic

import (
	"fmt"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"gorm.io/gorm"

	"gocms/core"
	"gocms/module/page/model"
)

// htmlSanitizer 用于过滤 body 字段中的危险 HTML 标签（如 <script>）
var htmlSanitizer = bluemonday.UGCPolicy()

// Logic page 业务逻辑
type Logic struct {
	db     *gorm.DB
	events core.EventBus
}

// NewLogic 创建 page 逻辑实例
func NewLogic(db *gorm.DB, events core.EventBus) *Logic {
	return &Logic{db: db, events: events}
}

// ---------------------------------------------------------------------------
// 管理端 CRUD
// ---------------------------------------------------------------------------

// List 页面列表（管理端，所有状态，分页）
// ownerID > 0 时仅返回该用户创建的页面（scope:own）
func (l *Logic) List(status string, page, pageSize int, ownerID int64) ([]model.Page, int64, error) {
	var items []model.Page
	var total int64

	query := l.db.Model(&model.Page{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if ownerID > 0 {
		query = query.Where("author_id = ?", ownerID)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("sort_order ASC, id DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	if items == nil {
		items = make([]model.Page, 0)
	}
	return items, total, err
}

// Create 创建页面
func (l *Logic) Create(title, slug, body, excerpt, featuredImage, template string, sortOrder int, meta model.PageMeta, seoTitle, seoDesc, seoKeywords string, authorID int64) (*model.Page, error) {
	// XSS 过滤：清除 body 中的危险标签
	body = htmlSanitizer.Sanitize(body)

	// 检查 slug 唯一
	var count int64
	l.db.Model(&model.Page{}).Where("slug = ?", slug).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("slug 已存在")
	}

	page := model.Page{
		Title:         title,
		Slug:          slug,
		Body:          body,
		Excerpt:       excerpt,
		Status:        model.PageStatusDraft,
		FeaturedImage: featuredImage,
		AuthorID:      authorID,
		Template:      template,
		SortOrder:     sortOrder,
		Meta:          meta,
		SeoTitle:      seoTitle,
		SeoDesc:       seoDesc,
		SeoKeywords:   seoKeywords,
	}

	if err := l.db.Create(&page).Error; err != nil {
		return nil, fmt.Errorf("创建页面失败: %w", err)
	}

	l.events.EmitAsync("page.created", core.ContentEvent{
		Module: "page",
		ID:     page.ID,
		Data:   page,
		UserID: authorID,
	})
	return &page, nil
}

// GetByID 按 ID 获取页面
func (l *Logic) GetByID(id int64) (*model.Page, error) {
	var page model.Page
	if err := l.db.First(&page, id).Error; err != nil {
		return nil, fmt.Errorf("页面不存在")
	}
	return &page, nil
}

// Update 更新页面
func (l *Logic) Update(id int64, title, slug, body, excerpt, featuredImage, template string, sortOrder *int, meta *model.PageMeta, seoTitle, seoDesc, seoKeywords *string, userID int64) error {
	var page model.Page
	if err := l.db.First(&page, id).Error; err != nil {
		return fmt.Errorf("页面不存在")
	}

	updates := map[string]interface{}{}
	if title != "" {
		updates["title"] = title
	}
	if slug != "" && slug != page.Slug {
		// 检查新 slug 唯一
		var count int64
		l.db.Model(&model.Page{}).Where("slug = ? AND id != ?", slug, id).Count(&count)
		if count > 0 {
			return fmt.Errorf("slug 已存在")
		}
		updates["slug"] = slug
	}
	if body != "" {
		// XSS 过滤：清除 body 中的危险标签
		updates["body"] = htmlSanitizer.Sanitize(body)
	}
	if excerpt != "" {
		updates["excerpt"] = excerpt
	}
	if featuredImage != "" {
		updates["featured_image"] = featuredImage
	}
	if template != "" {
		updates["template"] = template
	}
	if sortOrder != nil {
		updates["sort_order"] = *sortOrder
	}
	if meta != nil {
		updates["meta"] = *meta
	}
	if seoTitle != nil {
		updates["seo_title"] = *seoTitle
	}
	if seoDesc != nil {
		updates["seo_desc"] = *seoDesc
	}
	if seoKeywords != nil {
		updates["seo_keywords"] = *seoKeywords
	}

	if len(updates) == 0 {
		return nil
	}

	result := l.db.Model(&model.Page{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新失败: %w", result.Error)
	}

	l.events.EmitAsync("page.updated", core.ContentEvent{
		Module: "page",
		ID:     id,
		UserID: userID,
	})
	return nil
}

// Delete 删除页面（软删除）
func (l *Logic) Delete(id int64, userID int64) error {
	result := l.db.Delete(&model.Page{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("页面不存在")
	}

	l.events.EmitAsync("page.deleted", core.ContentEvent{
		Module: "page",
		ID:     id,
		UserID: userID,
	})
	return nil
}

// ---------------------------------------------------------------------------
// 发布 / 取消发布
// ---------------------------------------------------------------------------

// Publish 发布页面
func (l *Logic) Publish(id int64, userID int64) error {
	now := time.Now()
	result := l.db.Model(&model.Page{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       model.PageStatusPublished,
		"published_at": now,
	})
	if result.Error != nil {
		return fmt.Errorf("发布失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("页面不存在")
	}

	l.events.EmitAsync("page.published", core.ContentEvent{
		Module: "page",
		ID:     id,
		UserID: userID,
	})
	return nil
}

// Unpublish 取消发布（回到草稿状态）
func (l *Logic) Unpublish(id int64, userID int64) error {
	result := l.db.Model(&model.Page{}).Where("id = ?", id).Updates(map[string]interface{}{
		"status":       model.PageStatusDraft,
		"published_at": nil,
	})
	if result.Error != nil {
		return fmt.Errorf("取消发布失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("页面不存在")
	}

	l.events.EmitAsync("page.unpublished", core.ContentEvent{
		Module: "page",
		ID:     id,
		UserID: userID,
	})
	return nil
}

// ---------------------------------------------------------------------------
// 公开查询（仅 published）
// ---------------------------------------------------------------------------

// ListPublished 公开页面列表（仅 published，分页）
func (l *Logic) ListPublished(page, pageSize int) ([]model.Page, int64, error) {
	var items []model.Page
	var total int64

	query := l.db.Model(&model.Page{}).Where("status = ?", model.PageStatusPublished)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("sort_order ASC, published_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	if items == nil {
		items = make([]model.Page, 0)
	}
	return items, total, err
}

// GetPublishedByID 按 ID 获取已发布页面
func (l *Logic) GetPublishedByID(id int64) (*model.Page, error) {
	var page model.Page
	if err := l.db.Where("id = ? AND status = ?", id, model.PageStatusPublished).First(&page).Error; err != nil {
		return nil, fmt.Errorf("页面不存在")
	}
	return &page, nil
}

// GetPublishedBySlug 按 slug 获取已发布页面
func (l *Logic) GetPublishedBySlug(slug string) (*model.Page, error) {
	var page model.Page
	if err := l.db.Where("slug = ? AND status = ?", slug, model.PageStatusPublished).First(&page).Error; err != nil {
		return nil, fmt.Errorf("页面不存在")
	}
	return &page, nil
}
