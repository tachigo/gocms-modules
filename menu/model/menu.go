// Package model menu 数据模型
// MenuItem 支持树形结构，通过 parent_id 实现层级关系
package model

import (
	"time"

	"gorm.io/gorm"
)

// MenuItem 菜单项模型（树形结构）
type MenuItem struct {
	ID        int64          `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string         `gorm:"size:100;not null" json:"name"`                    // 菜单项名称
	Group     string         `gorm:"size:50;not null;index:idx_menu_group" json:"group"` // 菜单分组：如 main/footer/sidebar
	ParentID  *int64         `gorm:"index;default:null" json:"parent_id,omitempty"`    // 父菜单项ID，null表示根节点
	Order     int            `gorm:"not null;default:0" json:"order"`                  // 排序权重，越小越靠前
	URL       string         `gorm:"size:500;not null;default:''" json:"url"`          // 链接地址
	Icon      string         `gorm:"size:100;not null;default:''" json:"icon"`         // 图标类名（如 fa-home）
	Target    string         `gorm:"size:20;not null;default:'_self'" json:"target"`   // 打开方式：_self/_blank
	Status    string         `gorm:"size:20;not null;default:'active'" json:"status"`  // active/disabled
	CreatedAt time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// 关联子菜单项（非数据库字段）
	Children []*MenuItem `gorm:"-" json:"children,omitempty"`
}

func (MenuItem) TableName() string { return "menu_items" }

// MenuTree 菜单树节点
type MenuTree struct {
	ID       int64       `json:"id"`
	Name     string      `json:"name"`
	Group    string      `json:"group"`
	ParentID *int64      `json:"parent_id,omitempty"`
	Order    int         `json:"order"`
	URL      string      `json:"url"`
	Icon     string      `json:"icon"`
	Target   string      `json:"target"`
	Status   string      `json:"status"`
	Children []*MenuTree `json:"children,omitempty"`
}

// ToTree 将 MenuItem 转换为 MenuTree
func (item *MenuItem) ToTree() *MenuTree {
	return &MenuTree{
		ID:       item.ID,
		Name:     item.Name,
		Group:    item.Group,
		ParentID: item.ParentID,
		Order:    item.Order,
		URL:      item.URL,
		Icon:     item.Icon,
		Target:   item.Target,
		Status:   item.Status,
	}
}

// MenuGroup 菜单分组信息
type MenuGroup struct {
	Name  string `json:"name"`  // 分组标识
	Label string `json:"label"` // 分组显示名称
	Count int    `json:"count"` // 该分组下的菜单项数量
}
