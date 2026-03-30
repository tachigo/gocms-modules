// Package model taxonomy 数据模型
// 分类体系：词汇表（Vocabulary）+ 术语（Term）
// Vocabulary 是分类的容器（如"文章分类"、"标签"），Term 是具体的分类项
package model

import (
	"time"

	"gorm.io/gorm"
)

// Vocabulary 词汇表（分类容器）
// 例如：categories（文章分类）、tags（标签）、regions（地区）
type Vocabulary struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	MachineID   string         `gorm:"size:50;uniqueIndex;not null" json:"machine_id"`  // 机器名（URL 友好，如 categories）
	Name        string         `gorm:"size:100;not null" json:"name"`                   // 显示名称（如"文章分类"）
	Description string         `gorm:"size:500;not null;default:''" json:"description"`  // 描述
	Hierarchy   bool           `gorm:"not null;default:false" json:"hierarchy"`          // 是否支持层级（树形分类）
	Weight      int            `gorm:"not null;default:0" json:"weight"`                 // 排序权重（越大越前）
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Vocabulary) TableName() string { return "vocabularies" }

// Term 术语（分类项）
// 属于某个 Vocabulary，支持树形结构（通过 ParentID）
type Term struct {
	ID           int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	VocabularyID int64          `gorm:"index;not null" json:"vocabulary_id"`              // 所属词汇表
	ParentID     *int64         `gorm:"index" json:"parent_id"`                           // 父级术语 ID（层级分类用）
	Name         string         `gorm:"size:100;not null" json:"name"`                    // 术语名称
	Slug         string         `gorm:"size:100;not null" json:"slug"`                    // URL 友好标识
	Description  string         `gorm:"size:500;not null;default:''" json:"description"`   // 描述
	Weight       int            `gorm:"not null;default:0" json:"weight"`                  // 排序权重
	CreatedAt    time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
	Children     []Term         `gorm:"foreignKey:ParentID" json:"children,omitempty"`     // 子术语（查询时按需加载）
}

func (Term) TableName() string { return "terms" }
