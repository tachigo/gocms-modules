// Package model article 数据模型
// 文章内容模型 + 文章-分类/标签多对多关联表
package model

import (
	"time"

	"gorm.io/gorm"
)

// Article 文章模型
type Article struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string         `gorm:"size:200;not null" json:"title"`
	Slug        string         `gorm:"size:200;uniqueIndex;not null" json:"slug"`
	Summary     string         `gorm:"type:text" json:"summary"`
	Body        string         `gorm:"type:text;not null" json:"body"`
	CoverImage  *int64         `json:"cover_image"`                                    // FK → media.id
	AuthorID    int64          `gorm:"index;not null" json:"author_id"`                // FK → user.id
	Status      string         `gorm:"size:20;index;not null;default:'draft'" json:"status"` // draft | published | archived
	PublishedAt *time.Time     `gorm:"index" json:"published_at"`
	IsTop       bool           `gorm:"not null;default:false" json:"is_top"`
	SeoTitle    string         `gorm:"size:200;not null;default:''" json:"seo_title"`
	SeoDesc     string         `gorm:"size:500;not null;default:''" json:"seo_desc"`
	CreatedBy   int64          `gorm:"index;not null" json:"created_by"`
	UpdatedBy   *int64         `json:"updated_by"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`

	// 非持久化字段，查询后填充
	CategoryIDs []int64 `gorm:"-" json:"category_ids,omitempty"`
	TagIDs      []int64 `gorm:"-" json:"tag_ids,omitempty"`
}

func (Article) TableName() string { return "articles" }

// 文章状态常量
const (
	StatusDraft     = "draft"
	StatusPublished = "published"
	StatusArchived  = "archived"
)

// ArticleTaxonomy 文章-分类/标签关联表（多对多）
type ArticleTaxonomy struct {
	ArticleID int64  `gorm:"primaryKey" json:"article_id"`
	FieldID   string `gorm:"primaryKey;size:50" json:"field_id"` // "category" / "tag"
	TermID    int64  `gorm:"primaryKey" json:"term_id"`
}

func (ArticleTaxonomy) TableName() string { return "article_taxonomies" }
