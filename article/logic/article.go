// Package logic article 业务逻辑
// 文章 CRUD、发布状态管理、分类/标签关联管理
package logic

import (
	"fmt"
	"time"

	"github.com/microcosm-cc/bluemonday"
	"gorm.io/gorm"

	"gocms/core"
	"gocms/module/article/model"
)

// htmlSanitizer 用于过滤 body 字段中的危险 HTML 标签（如 <script>）
var htmlSanitizer = bluemonday.UGCPolicy()

// Logic 文章业务逻辑
type Logic struct {
	db     *gorm.DB
	events core.EventBus
}

// NewLogic 创建文章逻辑实例
func NewLogic(db *gorm.DB, events core.EventBus) *Logic {
	return &Logic{db: db, events: events}
}

// ---------------------------------------------------------------------------
// 公共 API（仅返回已发布文章）
// ---------------------------------------------------------------------------

// ListPublic 公开文章列表（仅 published）
func (l *Logic) ListPublic(page, pageSize int) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	query := l.db.Model(&model.Article{}).Where("status = ?", model.StatusPublished)
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("is_top DESC, published_at DESC").Offset(offset).Limit(pageSize).Find(&articles).Error
	if articles == nil {
		articles = make([]model.Article, 0)
	}

	// 加载关联的分类和标签
	l.loadTaxonomies(articles)

	return articles, total, err
}

// GetPublicByID 公开文章详情（仅 published）
func (l *Logic) GetPublicByID(id int64) (*model.Article, error) {
	var article model.Article
	err := l.db.Where("status = ? AND id = ?", model.StatusPublished, id).First(&article).Error
	if err != nil {
		return nil, fmt.Errorf("文章不存在")
	}
	l.loadTaxonomyForArticle(&article)
	return &article, nil
}

// GetPublicBySlug 通过 slug 获取公开文章
func (l *Logic) GetPublicBySlug(slug string) (*model.Article, error) {
	var article model.Article
	err := l.db.Where("status = ? AND slug = ?", model.StatusPublished, slug).First(&article).Error
	if err != nil {
		return nil, fmt.Errorf("文章不存在")
	}
	l.loadTaxonomyForArticle(&article)
	return &article, nil
}

// ---------------------------------------------------------------------------
// 管理 API（返回所有状态的文章）
// ---------------------------------------------------------------------------

// List 文章管理列表（全部状态，分页）
// ownerID > 0 时仅返回该用户创建的文章（scope:own）
func (l *Logic) List(status string, page, pageSize int, ownerID int64) ([]model.Article, int64, error) {
	var articles []model.Article
	var total int64

	query := l.db.Model(&model.Article{})
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if ownerID > 0 {
		query = query.Where("author_id = ?", ownerID)
	}
	query.Count(&total)

	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&articles).Error
	if articles == nil {
		articles = make([]model.Article, 0)
	}

	l.loadTaxonomies(articles)
	return articles, total, err
}

// GetByID 文章详情（管理端，所有状态）
func (l *Logic) GetByID(id int64) (*model.Article, error) {
	var article model.Article
	if err := l.db.First(&article, id).Error; err != nil {
		return nil, fmt.Errorf("文章不存在")
	}
	l.loadTaxonomyForArticle(&article)
	return &article, nil
}

// Create 创建文章
func (l *Logic) Create(article *model.Article, categoryIDs, tagIDs []int64) (*model.Article, error) {
	// XSS 过滤：清除 body 中的危险标签
	article.Body = htmlSanitizer.Sanitize(article.Body)

	// 检查 slug 唯一性
	if err := l.checkSlugUnique(article.Slug, 0); err != nil {
		return nil, err
	}

	// 设置初始状态
	if article.Status == "" {
		article.Status = model.StatusDraft
	}
	if article.Status == model.StatusPublished {
		now := time.Now()
		article.PublishedAt = &now
	}

	// 事务：创建文章 + 关联分类/标签
	err := l.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(article).Error; err != nil {
			return err
		}
		if err := l.saveTaxonomies(tx, article.ID, categoryIDs, tagIDs); err != nil {
			return err
		}
		return nil
	})

	// 捕获数据库唯一性冲突错误（竞态条件兜底）
	if err != nil && l.isDuplicateSlugError(err) {
		return nil, fmt.Errorf("文章别名(slug)已被使用")
	}

	if err != nil {
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}

	// 重新加载完整数据
	l.loadTaxonomyForArticle(article)

	// 发布事件
	l.events.EmitAsync("article.created", core.ContentEvent{
		Module: "article",
		ID:     article.ID,
		Data:   article,
		UserID: article.CreatedBy,
	})

	return article, nil
}

// Update 更新文章
func (l *Logic) Update(id int64, article *model.Article, categoryIDs, tagIDs []int64) (*model.Article, error) {
	// 获取旧数据
	oldArticle, err := l.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 检查 slug 唯一性
	if article.Slug != "" && article.Slug != oldArticle.Slug {
		if err := l.checkSlugUnique(article.Slug, id); err != nil {
			return nil, err
		}
	}

	// XSS 过滤：清除 body 中的危险标签
	if article.Body != "" {
		article.Body = htmlSanitizer.Sanitize(article.Body)
	}

	// 构建更新字段
	updates := map[string]interface{}{}
	if article.Title != "" {
		updates["title"] = article.Title
	}
	if article.Slug != "" {
		updates["slug"] = article.Slug
	}
	if article.Summary != "" || article.Body != "" {
		updates["summary"] = article.Summary
		updates["body"] = article.Body
	}
	if article.CoverImage != nil {
		updates["cover_image"] = *article.CoverImage
	}
	if article.AuthorID > 0 {
		updates["author_id"] = article.AuthorID
	}
	if article.SeoTitle != "" {
		updates["seo_title"] = article.SeoTitle
	}
	if article.SeoDesc != "" {
		updates["seo_desc"] = article.SeoDesc
	}
	if article.UpdatedBy != nil {
		updates["updated_by"] = *article.UpdatedBy
	}

	// 事务：更新文章 + 关联分类/标签
	err = l.db.Transaction(func(tx *gorm.DB) error {
		if len(updates) > 0 {
			if err := tx.Model(&model.Article{}).Where("id = ?", id).Updates(updates).Error; err != nil {
				return err
			}
		}
		if err := l.saveTaxonomies(tx, id, categoryIDs, tagIDs); err != nil {
			return err
		}
		return nil
	})

	// 捕获数据库唯一性冲突错误（竞态条件兜底）
	if err != nil && l.isDuplicateSlugError(err) {
		return nil, fmt.Errorf("文章别名(slug)已被使用")
	}

	if err != nil {
		return nil, fmt.Errorf("更新文章失败: %w", err)
	}

	// 重新加载完整数据
	updatedArticle, _ := l.GetByID(id)

	// 发布事件
	userID := int64(0)
	if article.UpdatedBy != nil {
		userID = *article.UpdatedBy
	}
	l.events.EmitAsync("article.updated", core.ContentEvent{
		Module:  "article",
		ID:      id,
		Data:    updatedArticle,
		OldData: oldArticle,
		UserID:  userID,
	})

	return updatedArticle, nil
}

// Delete 删除文章
func (l *Logic) Delete(id int64, userID int64) error {
	result := l.db.Delete(&model.Article{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("文章不存在")
	}

	// 清除分类/标签关联
	l.db.Where("article_id = ?", id).Delete(&model.ArticleTaxonomy{})

	// 发布事件
	l.events.EmitAsync("article.deleted", core.ContentEvent{
		Module: "article",
		ID:     id,
		UserID: userID,
	})

	return nil
}

// ---------------------------------------------------------------------------
// 发布状态管理
// ---------------------------------------------------------------------------

// Publish 发布文章
func (l *Logic) Publish(id int64, userID int64) error {
	var article model.Article
	if err := l.db.First(&article, id).Error; err != nil {
		return fmt.Errorf("文章不存在")
	}

	if article.Status == model.StatusPublished {
		return nil // 已经是发布状态
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":       model.StatusPublished,
		"published_at": now,
		"updated_by":   userID,
	}

	if err := l.db.Model(&article).Updates(updates).Error; err != nil {
		return fmt.Errorf("发布失败: %w", err)
	}

	// 重新加载完整数据
	l.db.First(&article, id)
	l.loadTaxonomyForArticle(&article)

	// 发布事件
	l.events.EmitAsync("article.published", core.ContentEvent{
		Module: "article",
		ID:     id,
		Data:   article,
		UserID: userID,
	})

	return nil
}

// Unpublish 取消发布（退回草稿）
func (l *Logic) Unpublish(id int64, userID int64) error {
	var article model.Article
	if err := l.db.First(&article, id).Error; err != nil {
		return fmt.Errorf("文章不存在")
	}

	if article.Status != model.StatusPublished {
		return nil // 不是发布状态
	}

	updates := map[string]interface{}{
		"status":     model.StatusDraft,
		"updated_by": userID,
	}

	if err := l.db.Model(&article).Updates(updates).Error; err != nil {
		return fmt.Errorf("取消发布失败: %w", err)
	}

	// 发布事件
	l.events.EmitAsync("article.archived", core.ContentEvent{
		Module: "article",
		ID:     id,
		UserID: userID,
	})

	return nil
}

// ---------------------------------------------------------------------------
// 内部辅助方法
// ---------------------------------------------------------------------------

// checkSlugUnique 检查 slug 是否唯一
// 先进行预检查，再依赖数据库唯一性索引作为兜底
func (l *Logic) checkSlugUnique(slug string, excludeID int64) error {
	var count int64
	query := l.db.Model(&model.Article{}).Where("slug = ?", slug)
	if excludeID > 0 {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return fmt.Errorf("检查 slug 唯一性失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("文章别名(slug)已被使用")
	}
	return nil
}

// isDuplicateSlugError 检查是否为数据库 slug 唯一性冲突错误
func (l *Logic) isDuplicateSlugError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	// SQLite 和 PostgreSQL 的唯一性冲突关键字
	return strings.Contains(errStr, "UNIQUE constraint failed") || 
	       strings.Contains(errStr, "unique constraint") ||
	       strings.Contains(errStr, "duplicate key")
}

// saveTaxonomies 保存文章-分类/标签关联
func (l *Logic) saveTaxonomies(tx *gorm.DB, articleID int64, categoryIDs, tagIDs []int64) error {
	// 删除旧关联
	if err := tx.Where("article_id = ?", articleID).Delete(&model.ArticleTaxonomy{}).Error; err != nil {
		return err
	}

	// 插入新关联
	var associations []model.ArticleTaxonomy

	for _, catID := range categoryIDs {
		associations = append(associations, model.ArticleTaxonomy{
			ArticleID: articleID,
			FieldID:   "category",
			TermID:    catID,
		})
	}

	for _, tagID := range tagIDs {
		associations = append(associations, model.ArticleTaxonomy{
			ArticleID: articleID,
			FieldID:   "tag",
			TermID:    tagID,
		})
	}

	if len(associations) > 0 {
		return tx.Create(&associations).Error
	}
	return nil
}

// loadTaxonomies 批量加载文章的分类和标签 ID
func (l *Logic) loadTaxonomies(articles []model.Article) {
	if len(articles) == 0 {
		return
	}

	articleIDs := make([]int64, len(articles))
	articleMap := make(map[int64]*model.Article)
	for i := range articles {
		articleIDs[i] = articles[i].ID
		articleMap[articles[i].ID] = &articles[i]
	}

	var associations []model.ArticleTaxonomy
	l.db.Where("article_id IN ?", articleIDs).Find(&associations)

	for _, assoc := range associations {
		if article, ok := articleMap[assoc.ArticleID]; ok {
			switch assoc.FieldID {
			case "category":
				article.CategoryIDs = append(article.CategoryIDs, assoc.TermID)
			case "tag":
				article.TagIDs = append(article.TagIDs, assoc.TermID)
			}
		}
	}
}

// loadTaxonomyForArticle 加载单篇文章的分类和标签 ID
func (l *Logic) loadTaxonomyForArticle(article *model.Article) {
	var associations []model.ArticleTaxonomy
	l.db.Where("article_id = ?", article.ID).Find(&associations)

	for _, assoc := range associations {
		switch assoc.FieldID {
		case "category":
			article.CategoryIDs = append(article.CategoryIDs, assoc.TermID)
		case "tag":
			article.TagIDs = append(article.TagIDs, assoc.TermID)
		}
	}
}
