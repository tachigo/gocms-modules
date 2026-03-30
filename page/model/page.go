// Package model page 数据模型
package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// PageStatus 页面状态
type PageStatus string

const (
	PageStatusDraft     PageStatus = "draft"
	PageStatusPublished PageStatus = "published"
)

// PageMeta SEO 元数据（存储为 JSON）
type PageMeta struct {
	MetaTitle       string `json:"meta_title,omitempty"`
	MetaDescription string `json:"meta_description,omitempty"`
	MetaKeywords    string `json:"meta_keywords,omitempty"`
	OgImage         string `json:"og_image,omitempty"`
}

// Scan 实现 sql.Scanner 接口（从数据库读取 JSON）
func (m *PageMeta) Scan(value interface{}) error {
	if value == nil {
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return fmt.Errorf("unsupported type for PageMeta: %T", value)
	}
	return json.Unmarshal(bytes, m)
}

// Value 实现 driver.Valuer 接口（写入数据库为 JSON）
func (m PageMeta) Value() (driver.Value, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

// Page 页面模型
type Page struct {
	ID             int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Title          string         `gorm:"size:255;not null" json:"title"`
	Slug           string         `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Body           string         `gorm:"type:text" json:"body"`
	Excerpt        string         `gorm:"size:500;not null;default:''" json:"excerpt"`
	Status         PageStatus     `gorm:"size:20;not null;default:'draft';index" json:"status"`
	FeaturedImage  string         `gorm:"size:500;not null;default:''" json:"featured_image"`
	AuthorID       int64          `gorm:"index;not null" json:"author_id"`
	Template       string         `gorm:"size:100;not null;default:''" json:"template"`
	SortOrder      int            `gorm:"not null;default:0" json:"sort_order"`
	Meta           PageMeta       `gorm:"type:text" json:"meta"`
	SeoTitle       string         `gorm:"size:200;not null;default:''" json:"seo_title"`
	SeoDesc        string         `gorm:"size:500;not null;default:''" json:"seo_desc"`
	SeoKeywords    string         `gorm:"size:500;not null;default:''" json:"seo_keywords"`
	PublishedAt    *time.Time     `gorm:"index" json:"published_at"`
	CreatedAt      time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Page) TableName() string { return "pages" }
