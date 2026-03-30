// Package controller menu API 控制器
// 菜单管理 API（ListGroups/GetTree/Create/Update/Delete/Reorder），GoFrame Bind 模式
package controller

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"gocms/internal/module/menu/logic"
	"gocms/internal/module/menu/model"
)

// ---------------------------------------------------------------------------
// Request / Response 定义
// ---------------------------------------------------------------------------

// --- 获取菜单分组列表 ---

type ListGroupsReq struct {
	g.Meta `path:"/menu-groups" method:"GET" tags:"菜单管理" summary:"菜单分组列表" dc:"获取所有菜单分组及其菜单项数量"`
}

type ListGroupsRes struct {
	g.Meta `mime:"application/json"`
	List   []model.MenuGroup `json:"list" dc:"分组列表"`
}

// --- 获取菜单树（管理员） ---

type GetTreeReq struct {
	g.Meta `path:"/menus/{group}/tree" method:"GET" tags:"菜单管理" summary:"获取菜单树" dc:"获取指定分组的完整菜单树（包含禁用项）"`
	Group  string `json:"group" in:"path" v:"required" dc:"菜单分组，如 main/footer"`
}

type GetTreeRes struct {
	g.Meta `mime:"application/json"`
	Tree   []*model.MenuTree `json:"tree" dc:"菜单树结构"`
}

// --- 创建菜单项 ---

type CreateMenuReq struct {
	g.Meta   `path:"/menus" method:"POST" tags:"菜单管理" summary:"创建菜单项" dc:"创建新的菜单项"`
	Name     string `json:"name" v:"required|max-length:100" dc:"菜单项名称"`
	Group    string `json:"group" v:"required|max-length:50" dc:"菜单分组，如 main/footer"`
	ParentID *int64 `json:"parent_id,omitempty" dc:"父菜单项ID，为空表示根节点"`
	Order    int    `json:"order" d:"0" dc:"排序权重，越小越靠前"`
	URL      string `json:"url" v:"required|max-length:500" dc:"链接地址"`
	Icon     string `json:"icon" dc:"图标类名"`
	Target   string `json:"target" v:"in:_self,_blank" d:"_self" dc:"打开方式"`
	Status   string `json:"status" v:"in:active,disabled" d:"active" dc:"状态"`
}

type CreateMenuRes struct {
	g.Meta `mime:"application/json"`
	ID     int64 `json:"id" dc:"菜单项ID"`
}

// --- 获取菜单项详情 ---

type GetMenuReq struct {
	g.Meta `path:"/menus/{id}" method:"GET" tags:"菜单管理" summary:"菜单项详情" dc:"获取指定菜单项信息"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"菜单项ID"`
}

type GetMenuRes struct {
	g.Meta `mime:"application/json"`
	*model.MenuItem
}

// --- 更新菜单项 ---

type UpdateMenuReq struct {
	g.Meta   `path:"/menus/{id}" method:"PUT" tags:"菜单管理" summary:"更新菜单项" dc:"更新菜单项信息"`
	ID       int64  `json:"id" in:"path" v:"required|min:1" dc:"菜单项ID"`
	Name     string `json:"name" v:"required|max-length:100" dc:"菜单项名称"`
	ParentID *int64 `json:"parent_id,omitempty" dc:"父菜单项ID，设0表示移除父节点"`
	Order    int    `json:"order" dc:"排序权重"`
	URL      string `json:"url" v:"required|max-length:500" dc:"链接地址"`
	Icon     string `json:"icon" dc:"图标类名"`
	Target   string `json:"target" v:"in:_self,_blank" dc:"打开方式"`
	Status   string `json:"status" v:"in:active,disabled" dc:"状态"`
}

type UpdateMenuRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除菜单项 ---

type DeleteMenuReq struct {
	g.Meta `path:"/menus/{id}" method:"DELETE" tags:"菜单管理" summary:"删除菜单项" dc:"删除菜单项及其子菜单"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"菜单项ID"`
}

type DeleteMenuRes struct {
	g.Meta `mime:"application/json"`
}

// --- 批量排序 ---

type ReorderMenusReq struct {
	g.Meta `path:"/menus/{group}/reorder" method:"PUT" tags:"菜单管理" summary:"批量排序" dc:"批量更新菜单项排序权重"`
	Group  string         `json:"group" in:"path" v:"required" dc:"菜单分组"`
	Orders map[int64]int  `json:"orders" v:"required" dc:"排序映射，key为菜单项ID，value为排序值"`
}

type ReorderMenusRes struct {
	g.Meta `mime:"application/json"`
}

// --- 移动菜单项 ---

type MoveMenuReq struct {
	g.Meta       `path:"/menus/{id}/move" method:"PUT" tags:"菜单管理" summary:"移动菜单项" dc:"移动菜单项到新的父节点或分组"`
	ID           int64   `json:"id" in:"path" v:"required|min:1" dc:"菜单项ID"`
	NewParentID  *int64  `json:"new_parent_id,omitempty" dc:"新的父菜单项ID"`
	NewGroup     string  `json:"new_group" v:"required" dc:"新的菜单分组"`
}

type MoveMenuRes struct {
	g.Meta `mime:"application/json"`
}

// ---------------------------------------------------------------------------
// Public API Request / Response
// ---------------------------------------------------------------------------

// --- 获取公开菜单树 ---

type GetPublicMenuTreeReq struct {
	g.Meta `path:"/menus/{group}" method:"GET" tags:"菜单" summary:"获取菜单树（公开）" dc:"获取指定分组的公开菜单树，仅返回激活状态的菜单项"`
	Group  string `json:"group" in:"path" v:"required" dc:"菜单分组，如 main/footer"`
}

type GetPublicMenuTreeRes struct {
	g.Meta `mime:"application/json"`
	Tree   []*model.MenuTree `json:"tree" dc:"菜单树结构"`
}

// ---------------------------------------------------------------------------
// Admin Controller
// ---------------------------------------------------------------------------

// AdminController 菜单管理控制器（/api/admin/*）
type AdminController struct {
	logic *logic.Logic
}

// NewAdminController 创建菜单管理控制器
func NewAdminController(l *logic.Logic) *AdminController {
	return &AdminController{logic: l}
}

// ListGroups 获取菜单分组列表
func (c *AdminController) ListGroups(ctx context.Context, req *ListGroupsReq) (res *ListGroupsRes, err error) {
	groups, err := c.logic.ListGroups()
	if err != nil {
		return nil, err
	}
	return &ListGroupsRes{List: groups}, nil
}

// GetTree 获取菜单树
func (c *AdminController) GetTree(ctx context.Context, req *GetTreeReq) (res *GetTreeRes, err error) {
	tree, err := c.logic.GetTreeAll(req.Group)
	if err != nil {
		return nil, err
	}
	return &GetTreeRes{Tree: tree}, nil
}

// CreateMenu 创建菜单项
func (c *AdminController) CreateMenu(ctx context.Context, req *CreateMenuReq) (res *CreateMenuRes, err error) {
	item, err := c.logic.Create(req.Name, req.Group, req.ParentID, req.Order, req.URL, req.Icon, req.Target, req.Status)
	if err != nil {
		return nil, err
	}
	return &CreateMenuRes{ID: item.ID}, nil
}

// GetMenu 获取菜单项详情
func (c *AdminController) GetMenu(ctx context.Context, req *GetMenuReq) (res *GetMenuRes, err error) {
	item, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	return &GetMenuRes{MenuItem: item}, nil
}

// UpdateMenu 更新菜单项
func (c *AdminController) UpdateMenu(ctx context.Context, req *UpdateMenuReq) (res *UpdateMenuRes, err error) {
	// 处理 parent_id = 0 的情况（表示移除父节点）
	var parentID *int64
	if req.ParentID != nil && *req.ParentID > 0 {
		parentID = req.ParentID
	}

	if err := c.logic.Update(req.ID, req.Name, parentID, req.Order, req.URL, req.Icon, req.Target, req.Status); err != nil {
		return nil, err
	}
	return &UpdateMenuRes{}, nil
}

// DeleteMenu 删除菜单项
func (c *AdminController) DeleteMenu(ctx context.Context, req *DeleteMenuReq) (res *DeleteMenuRes, err error) {
	if err := c.logic.Delete(req.ID); err != nil {
		return nil, err
	}
	return &DeleteMenuRes{}, nil
}

// ReorderMenus 批量排序
func (c *AdminController) ReorderMenus(ctx context.Context, req *ReorderMenusReq) (res *ReorderMenusRes, err error) {
	if err := c.logic.Reorder(req.Group, req.Orders); err != nil {
		return nil, err
	}
	return &ReorderMenusRes{}, nil
}

// MoveMenu 移动菜单项
func (c *AdminController) MoveMenu(ctx context.Context, req *MoveMenuReq) (res *MoveMenuRes, err error) {
	var parentID *int64
	if req.NewParentID != nil && *req.NewParentID > 0 {
		parentID = req.NewParentID
	}

	if err := c.logic.Move(req.ID, parentID, req.NewGroup); err != nil {
		return nil, err
	}
	return &MoveMenuRes{}, nil
}

// ---------------------------------------------------------------------------
// Public Controller
// ---------------------------------------------------------------------------

// PublicController 公开菜单控制器（/api/*）
type PublicController struct {
	logic *logic.Logic
}

// NewPublicController 创建公开菜单控制器
func NewPublicController(l *logic.Logic) *PublicController {
	return &PublicController{logic: l}
}

// GetPublicMenuTree 获取公开菜单树（仅返回激活状态的菜单项）
func (c *PublicController) GetPublicMenuTree(ctx context.Context, req *GetPublicMenuTreeReq) (res *GetPublicMenuTreeRes, err error) {
	tree, err := c.logic.GetTree(req.Group)
	if err != nil {
		return nil, err
	}
	return &GetPublicMenuTreeRes{Tree: tree}, nil
}
