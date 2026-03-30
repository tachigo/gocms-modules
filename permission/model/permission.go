// Package model permission 数据模型
// Role 和 Permission 数据模型定义
package model

import (
	"time"
)

// Role 角色模型
type Role struct {
	ID          int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	Name        string    `gorm:"size:50;uniqueIndex;not null" json:"name"`            // admin / editor / viewer / custom_xxx
	Label       string    `gorm:"size:100;not null" json:"label"`                      // 显示名称：管理员 / 编辑 / 访客
	Description string    `gorm:"type:text;not null;default:''" json:"description"`    // 角色描述
	IsSystem    bool      `gorm:"not null;default:false" json:"is_system"`             // 系统内置角色不可删除
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Role) TableName() string { return "roles" }

// Permission 权限模型
// 表示某个角色对某个模块的某个操作拥有的权限
type Permission struct {
	ID        int64     `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID    int64     `gorm:"index;not null" json:"role_id"`                       // 角色ID
	Module    string    `gorm:"size:50;not null" json:"module"`                      // 模块名，如 user / article / media
	Action    string    `gorm:"size:20;not null" json:"action"`                      // 操作：create / read / update / delete / manage
	Scope     string    `gorm:"size:20;not null;default:'all'" json:"scope"`         // 数据范围：own / all
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Permission) TableName() string { return "permissions" }

// UserRole 用户角色关联模型（多对多）
type UserRole struct {
	UserID int64 `gorm:"primaryKey" json:"user_id"`
	RoleID int64 `gorm:"primaryKey" json:"role_id"`
}

func (UserRole) TableName() string { return "user_roles" }

// RoleWithPermissions 角色及其权限列表（查询用）
type RoleWithPermissions struct {
	Role
	Permissions []Permission `json:"permissions"`
}

// UserWithRoles 用户及其角色列表（查询用）
type UserWithRoles struct {
	UserID int64  `json:"user_id"`
	RoleIDs []int64 `json:"role_ids"`
	Roles []Role  `json:"roles"`
}
