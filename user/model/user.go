// Package model user 数据模型
package model

import (
	"time"

	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string         `gorm:"size:50;uniqueIndex;not null" json:"username"`
	Email     string         `gorm:"size:100;uniqueIndex;not null" json:"email"`
	Password  string         `gorm:"size:255;not null" json:"-"`                    // bcrypt hash，永不返回给前端
	Nickname  string         `gorm:"size:50;not null;default:''" json:"nickname"`
	Avatar    string         `gorm:"size:500;not null;default:''" json:"avatar"`
	Status    string         `gorm:"size:20;not null;default:'active'" json:"status"` // active | disabled
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (User) TableName() string { return "users" }
