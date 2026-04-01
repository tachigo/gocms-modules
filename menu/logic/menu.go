// Package logic menu 业务逻辑
// 菜单 CRUD + 树形结构管理（获取树、移动节点、排序）
package logic

import (
	"fmt"
	"sort"

	"gorm.io/gorm"

	"gocms/core"
	"gocms/module/menu/model"
)

// Logic menu 业务逻辑
type Logic struct {
	db     *gorm.DB
	events core.EventBus
}

// NewLogic 创建 menu 逻辑实例
func NewLogic(db *gorm.DB, events core.EventBus) *Logic {
	return &Logic{db: db, events: events}
}

// ---------------------------------------------------------------------------
// 菜单分组管理
// ---------------------------------------------------------------------------

// ListGroups 获取所有菜单分组列表
func (l *Logic) ListGroups() ([]model.MenuGroup, error) {
	var groups []struct {
		Group string
		Count int64
	}

	if err := l.db.Model(&model.MenuItem{}).
		Select("`group`, COUNT(*) as count").
		Group("`group`").
		Find(&groups).Error; err != nil {
		return nil, err
	}

	result := make([]model.MenuGroup, 0, len(groups))
	for _, g := range groups {
		result = append(result, model.MenuGroup{
			Name:  g.Group,
			Label: l.getGroupLabel(g.Group),
			Count: int(g.Count),
		})
	}

	// 如果没有分组，返回默认分组
	if len(result) == 0 {
		result = []model.MenuGroup{
			{Name: "main", Label: "主导航", Count: 0},
			{Name: "footer", Label: "页脚导航", Count: 0},
		}
	}

	return result, nil
}

// getGroupLabel 获取分组显示名称
func (l *Logic) getGroupLabel(name string) string {
	labels := map[string]string{
		"main":     "主导航",
		"footer":   "页脚导航",
		"sidebar":  "侧边栏",
		"user":     "用户中心",
		"admin":    "后台管理",
		"mobile":   "移动端导航",
	}
	if label, ok := labels[name]; ok {
		return label
	}
	return name
}

// ---------------------------------------------------------------------------
// 菜单树管理
// ---------------------------------------------------------------------------

// GetTree 获取指定分组的菜单树
func (l *Logic) GetTree(group string) ([]*model.MenuTree, error) {
	// 检查分组是否存在
	var count int64
	if err := l.db.Model(&model.MenuItem{}).Where("`group` = ?", group).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("菜单分组 '%s' 不存在", group)
	}

	var items []model.MenuItem
	if err := l.db.Where("`group` = ? AND status = ?", group, "active").
		Order("`order` ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}

	return l.buildTree(items), nil
}

// GetTreeAll 获取指定分组的完整菜单树（包含禁用项，管理后台用）
func (l *Logic) GetTreeAll(group string) ([]*model.MenuTree, error) {
	// 检查分组是否存在
	var count int64
	if err := l.db.Model(&model.MenuItem{}).Where("`group` = ?", group).Count(&count).Error; err != nil {
		return nil, err
	}
	if count == 0 {
		return nil, fmt.Errorf("菜单分组 '%s' 不存在", group)
	}

	var items []model.MenuItem
	if err := l.db.Where("`group` = ?", group).
		Order("`order` ASC, id ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}

	return l.buildTree(items), nil
}

// GetTreeWithDepth 获取指定分组的菜单树，限制深度
func (l *Logic) GetTreeWithDepth(group string, maxDepth int) ([]*model.MenuTree, error) {
	tree, err := l.GetTree(group)
	if err != nil {
		return nil, err
	}

	if maxDepth > 0 {
		l.trimTreeDepth(tree, 1, maxDepth)
	}

	return tree, nil
}

// buildTree 将扁平列表构建为树形结构
func (l *Logic) buildTree(items []model.MenuItem) []*model.MenuTree {
	// 创建ID映射表
	itemMap := make(map[int64]*model.MenuTree)
	for i := range items {
		itemMap[items[i].ID] = items[i].ToTree()
	}

	// 构建树
	var roots []*model.MenuTree
	for i := range items {
		treeNode := itemMap[items[i].ID]
		if items[i].ParentID == nil {
			// 根节点
			roots = append(roots, treeNode)
		} else {
			// 子节点
			if parent, ok := itemMap[*items[i].ParentID]; ok {
				parent.Children = append(parent.Children, treeNode)
			} else {
				// 父节点不存在，作为根节点处理
				roots = append(roots, treeNode)
			}
		}
	}

	// 递归排序每个节点的子节点
	l.sortTree(roots)

	return roots
}

// sortTree 递归排序树节点
func (l *Logic) sortTree(nodes []*model.MenuTree) {
	sort.Slice(nodes, func(i, j int) bool {
		if nodes[i].Order != nodes[j].Order {
			return nodes[i].Order < nodes[j].Order
		}
		return nodes[i].ID < nodes[j].ID
	})

	for _, node := range nodes {
		if len(node.Children) > 0 {
			l.sortTree(node.Children)
		}
	}
}

// trimTreeDepth 修剪树深度
func (l *Logic) trimTreeDepth(nodes []*model.MenuTree, currentDepth, maxDepth int) {
	if currentDepth >= maxDepth {
		for _, node := range nodes {
			node.Children = nil
		}
		return
	}

	for _, node := range nodes {
		if len(node.Children) > 0 {
			l.trimTreeDepth(node.Children, currentDepth+1, maxDepth)
		}
	}
}

// ---------------------------------------------------------------------------
// 菜单项 CRUD
// ---------------------------------------------------------------------------

// Create 创建菜单项
func (l *Logic) Create(name, group string, parentID *int64, order int, url, icon, target, status string) (*model.MenuItem, error) {
	// 验证父节点
	if parentID != nil {
		var parent model.MenuItem
		if err := l.db.First(&parent, *parentID).Error; err != nil {
			return nil, fmt.Errorf("父菜单项不存在")
		}
		if parent.Group != group {
			return nil, fmt.Errorf("父菜单项必须在同一分组")
		}
	}

	item := model.MenuItem{
		Name:     name,
		Group:    group,
		ParentID: parentID,
		Order:    order,
		URL:      url,
		Icon:     icon,
		Target:   target,
		Status:   status,
	}

	if item.Target == "" {
		item.Target = "_self"
	}
	if item.Status == "" {
		item.Status = "active"
	}

	if err := l.db.Create(&item).Error; err != nil {
		return nil, fmt.Errorf("创建菜单项失败: %w", err)
	}

	l.events.EmitAsync("menu.created", core.MenuEvent{MenuID: fmt.Sprintf("%d", item.ID)})
	return &item, nil
}

// GetByID 根据ID获取菜单项
func (l *Logic) GetByID(id int64) (*model.MenuItem, error) {
	var item model.MenuItem
	if err := l.db.First(&item, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("菜单项不存在")
		}
		return nil, err
	}
	return &item, nil
}

// Update 更新菜单项
func (l *Logic) Update(id int64, name string, parentID *int64, order int, url, icon, target, status string) error {
	var item model.MenuItem
	if err := l.db.First(&item, id).Error; err != nil {
		return fmt.Errorf("菜单项不存在")
	}

	// 验证父节点（不能将自己或子节点设为父节点）
	if parentID != nil && *parentID != 0 {
		if *parentID == id {
			return fmt.Errorf("不能将菜单项设为自己的父节点")
		}
		// 检查是否将子节点设为父节点（避免循环引用）
		if l.isDescendant(id, *parentID) {
			return fmt.Errorf("不能将子节点设为父节点")
		}

		var parent model.MenuItem
		if err := l.db.First(&parent, *parentID).Error; err != nil {
			return fmt.Errorf("父菜单项不存在")
		}
		if parent.Group != item.Group {
			return fmt.Errorf("父菜单项必须在同一分组")
		}
	}

	updates := map[string]interface{}{
		"name":   name,
		"order":  order,
		"url":    url,
		"icon":   icon,
		"target": target,
		"status": status,
	}

	if parentID != nil && *parentID != 0 {
		updates["parent_id"] = *parentID
	} else {
		updates["parent_id"] = nil
	}

	if err := l.db.Model(&item).Updates(updates).Error; err != nil {
		return fmt.Errorf("更新菜单项失败: %w", err)
	}

	l.events.EmitAsync("menu.updated", core.MenuEvent{MenuID: fmt.Sprintf("%d", id)})
	return nil
}

// Delete 删除菜单项（软删除）
// 如果菜单项有子节点，会一并删除
func (l *Logic) Delete(id int64) error {
	var item model.MenuItem
	if err := l.db.First(&item, id).Error; err != nil {
		return fmt.Errorf("菜单项不存在")
	}

	// 递归删除所有子节点
	if err := l.deleteChildren(id); err != nil {
		return err
	}

	// 删除当前节点
	if err := l.db.Delete(&item).Error; err != nil {
		return fmt.Errorf("删除菜单项失败: %w", err)
	}

	l.events.EmitAsync("menu.deleted", core.MenuEvent{MenuID: fmt.Sprintf("%d", id)})
	return nil
}

// deleteChildren 递归删除子节点
func (l *Logic) deleteChildren(parentID int64) error {
	var children []model.MenuItem
	if err := l.db.Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return err
	}

	for _, child := range children {
		if err := l.deleteChildren(child.ID); err != nil {
			return err
		}
		if err := l.db.Delete(&child).Error; err != nil {
			return err
		}
	}

	return nil
}

// isDescendant 检查 targetID 是否是 sourceID 的后代节点
func (l *Logic) isDescendant(sourceID, targetID int64) bool {
	var children []model.MenuItem
	if err := l.db.Where("parent_id = ?", sourceID).Find(&children).Error; err != nil {
		return false
	}

	for _, child := range children {
		if child.ID == targetID {
			return true
		}
		if l.isDescendant(child.ID, targetID) {
			return true
		}
	}

	return false
}

// ---------------------------------------------------------------------------
// 排序与移动
// ---------------------------------------------------------------------------

// Reorder 批量排序菜单项
// orders: map[菜单项ID]排序值
func (l *Logic) Reorder(group string, orders map[int64]int) error {
	return l.db.Transaction(func(tx *gorm.DB) error {
		for id, order := range orders {
			// 验证菜单项是否属于该分组
			var item model.MenuItem
			if err := tx.Where("id = ? AND `group` = ?", id, group).First(&item).Error; err != nil {
				if err == gorm.ErrRecordNotFound {
					continue // 跳过不存在的或不在该分组的
				}
				return err
			}

			if err := tx.Model(&item).Update("order", order).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// Move 移动菜单项到新的父节点或分组
func (l *Logic) Move(id int64, newParentID *int64, newGroup string) error {
	var item model.MenuItem
	if err := l.db.First(&item, id).Error; err != nil {
		return fmt.Errorf("菜单项不存在")
	}

	// 检查循环引用
	if newParentID != nil && *newParentID != 0 {
		if *newParentID == id {
			return fmt.Errorf("不能将菜单项设为自己的父节点")
		}
		if l.isDescendant(id, *newParentID) {
			return fmt.Errorf("不能将子节点设为父节点")
		}

		var parent model.MenuItem
		if err := l.db.First(&parent, *newParentID).Error; err != nil {
			return fmt.Errorf("父菜单项不存在")
		}
		if parent.Group != newGroup {
			return fmt.Errorf("父菜单项必须在同一分组")
		}
	}

	updates := map[string]interface{}{
		"group": newGroup,
	}

	if newParentID != nil && *newParentID != 0 {
		updates["parent_id"] = *newParentID
	} else {
		updates["parent_id"] = nil
	}

	if err := l.db.Model(&item).Updates(updates).Error; err != nil {
		return fmt.Errorf("移动菜单项失败: %w", err)
	}

	// 如果分组改变，需要更新所有子节点的分组
	if newGroup != item.Group {
		if err := l.updateChildrenGroup(id, newGroup); err != nil {
			return err
		}
	}

	l.events.EmitAsync("menu.moved", core.MenuEvent{MenuID: fmt.Sprintf("%d", id)})
	return nil
}

// updateChildrenGroup 递归更新子节点的分组
func (l *Logic) updateChildrenGroup(parentID int64, newGroup string) error {
	var children []model.MenuItem
	if err := l.db.Where("parent_id = ?", parentID).Find(&children).Error; err != nil {
		return err
	}

	for _, child := range children {
		if err := l.db.Model(&child).Update("group", newGroup).Error; err != nil {
			return err
		}
		if err := l.updateChildrenGroup(child.ID, newGroup); err != nil {
			return err
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// 初始化
// ---------------------------------------------------------------------------

// InitDefaultMenus 初始化默认菜单（首次启动时）
func (l *Logic) InitDefaultMenus() error {
	var count int64
	if err := l.db.Model(&model.MenuItem{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有菜单，跳过
	}

	// 创建主导航默认菜单
	defaults := []model.MenuItem{
		{Name: "首页", Group: "main", Order: 1, URL: "/", Status: "active"},
		{Name: "关于我们", Group: "main", Order: 2, URL: "/about", Status: "active"},
		{Name: "联系我们", Group: "main", Order: 3, URL: "/contact", Status: "active"},
	}

	for _, item := range defaults {
		if err := l.db.Create(&item).Error; err != nil {
			return err
		}
	}

	return nil
}
