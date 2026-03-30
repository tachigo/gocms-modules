// Package model media 数据模型
package model

import (
	"time"

	"gorm.io/gorm"
)

// Media 媒体文件
type Media struct {
	ID          int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	FolderID    *int64         `gorm:"index" json:"folder_id"`
	Filename    string         `gorm:"size:255;not null" json:"filename"`        // 原始文件名
	StoragePath string         `gorm:"size:500;not null" json:"storage_path"`    // 磁盘存储路径
	URL         string         `gorm:"-" json:"url"`                             // 可访问 URL（查询后填充）
	MimeType    string         `gorm:"size:100;not null;index" json:"mime_type"`
	Size        int64          `gorm:"not null" json:"size"`                     // 字节数
	Width       *int           `json:"width,omitempty"`
	Height      *int           `json:"height,omitempty"`
	Alt         string         `gorm:"size:255;not null;default:''" json:"alt"`
	Title       string         `gorm:"size:255;not null;default:''" json:"title"`
	UploadedBy  int64          `gorm:"index;not null" json:"uploaded_by"`
	CreatedAt   time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Media) TableName() string { return "media" }

// MediaFolder 媒体文件夹
type MediaFolder struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	ParentID  *int64         `gorm:"index" json:"parent_id"`
	Sort      int            `gorm:"not null;default:0" json:"sort"`
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	Children  []MediaFolder  `gorm:"foreignKey:ParentID" json:"children,omitempty"`
}

func (MediaFolder) TableName() string { return "media_folders" }

// FillURL 填充可访问的 URL（StoragePath 就是 web 路径）
func (m *Media) FillURL() {
	m.URL = m.StoragePath
}
