// Package controller permission 管理 API 控制器
// 角色管理 API（/api/admin/roles）
package controller

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/module/permission/logic"
	"gocms/module/permission/model"
)

// ---------------------------------------------------------------------------
// Request / Response 定义
// ---------------------------------------------------------------------------

// --- 角色列表 ---

type ListRolesReq struct {
	g.Meta `path:"/roles" method:"GET" tags:"角色管理" summary:"角色列表" dc:"获取所有角色列表"`
}

type ListRolesRes struct {
	g.Meta `mime:"application/json"`
	List   interface{} `json:"list" dc:"角色列表"`
}

// --- 角色详情 ---

type GetRoleReq struct {
	g.Meta `path:"/roles/{id}" method:"GET" tags:"角色管理" summary:"角色详情" dc:"获取指定角色详情及权限列表"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"角色ID"`
}

type GetRoleRes struct {
	g.Meta `mime:"application/json"`
	*RoleDetail
}

type RoleDetail struct {
	ID          int64              `json:"id"`
	Name        string             `json:"name"`
	Label       string             `json:"label"`
	Description string             `json:"description"`
	IsSystem    bool               `json:"is_system"`
	Permissions []PermissionDetail `json:"permissions"`
	CreatedAt   string             `json:"created_at"`
}

type PermissionDetail struct {
	ID     int64  `json:"id"`
	Module string `json:"module"`
	Action string `json:"action"`
	Scope  string `json:"scope"`
}

// --- 创建角色 ---

type CreateRoleReq struct {
	g.Meta      `path:"/roles" method:"POST" tags:"角色管理" summary:"创建角色" dc:"创建自定义角色并分配权限"`
	Name        string                `json:"name" v:"required|min-length:2|max-length:50|regex:^[a-z0-9_]+$" dc:"角色标识（英文小写+下划线）"`
	Label       string                `json:"label" v:"required|max-length:100" dc:"角色显示名称"`
	Description string                `json:"description" dc:"角色描述"`
	Permissions []PermissionInput     `json:"permissions" dc:"权限列表"`
}

type PermissionInput struct {
	Module string `json:"module" v:"required" dc:"模块名"`
	Action string `json:"action" v:"required" dc:"操作：create/read/update/delete/manage"`
	Scope  string `json:"scope" v:"in:own,all" d:"all" dc:"数据范围：own/all"`
}

type CreateRoleRes struct {
	g.Meta `mime:"application/json"`
	ID     int64 `json:"id" dc:"角色ID"`
}

// --- 编辑角色 ---

type UpdateRoleReq struct {
	g.Meta      `path:"/roles/{id}" method:"PUT" tags:"角色管理" summary:"编辑角色" dc:"更新角色信息和权限"`
	ID          int64             `json:"id" in:"path" v:"required|min:1" dc:"角色ID"`
	Label       string            `json:"label" dc:"角色显示名称"`
	Description string            `json:"description" dc:"角色描述"`
	Permissions []PermissionInput `json:"permissions" dc:"权限列表（全量替换）"`
}

type UpdateRoleRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除角色 ---

type DeleteRoleReq struct {
	g.Meta `path:"/roles/{id}" method:"DELETE" tags:"角色管理" summary:"删除角色" dc:"删除自定义角色（系统角色不能删除）"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"角色ID"`
}

type DeleteRoleRes struct {
	g.Meta `mime:"application/json"`
}

// --- 获取所有可用权限 ---

type AvailablePermissionsReq struct {
	g.Meta `path:"/permissions/available" method:"GET" tags:"角色管理" summary:"可用权限列表" dc:"获取所有模块声明的权限定义"`
}

type AvailablePermissionsRes struct {
	g.Meta `mime:"application/json"`
	Groups []PermissionGroupDetail `json:"groups" dc:"按模块分组的权限列表"`
}

type PermissionGroupDetail struct {
	Module      string             `json:"module"`
	Permissions []PermissionDefDetail `json:"permissions"`
}

type PermissionDefDetail struct {
	Action      string   `json:"action"`
	Description string   `json:"description"`
	Scopes      []string `json:"scopes"`
}

// --- 为用户分配角色 ---

type AssignRolesReq struct {
	g.Meta  `path:"/users/{user_id}/roles" method:"PUT" tags:"角色管理" summary:"为用户分配角色" dc:"全量替换用户的角色"`
	UserID  int64   `json:"user_id" in:"path" v:"required|min:1" dc:"用户ID"`
	RoleIDs []int64 `json:"role_ids" dc:"角色ID列表"`
}

type AssignRolesRes struct {
	g.Meta `mime:"application/json"`
}

// --- 获取用户角色 ---

type GetUserRolesReq struct {
	g.Meta `path:"/users/{user_id}/roles" method:"GET" tags:"角色管理" summary:"获取用户角色" dc:"获取指定用户的角色列表"`
	UserID int64 `json:"user_id" in:"path" v:"required|min:1" dc:"用户ID"`
}

type GetUserRolesRes struct {
	g.Meta `mime:"application/json"`
	Roles  []RoleSimple `json:"roles"`
}

type RoleSimple struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Label string `json:"label"`
}

// ---------------------------------------------------------------------------
// Permission Controller
// ---------------------------------------------------------------------------

// PermissionController 权限管理控制器（/api/admin/roles）
type PermissionController struct {
	logic *logic.PermissionLogic
}

// NewPermissionController 创建权限管理控制器
func NewPermissionController(l *logic.PermissionLogic) *PermissionController {
	return &PermissionController{logic: l}
}

// ListRoles 角色列表
func (c *PermissionController) ListRoles(ctx context.Context, req *ListRolesReq) (res *ListRolesRes, err error) {
	roles, err := c.logic.ListRoles()
	if err != nil {
		return nil, err
	}
	return &ListRolesRes{List: roles}, nil
}

// GetRole 角色详情
func (c *PermissionController) GetRole(ctx context.Context, req *GetRoleReq) (res *GetRoleRes, err error) {
	roleWithPerms, err := c.logic.GetRole(req.ID)
	if err != nil {
		return nil, err
	}

	perms := make([]PermissionDetail, 0, len(roleWithPerms.Permissions))
	for _, p := range roleWithPerms.Permissions {
		perms = append(perms, PermissionDetail{
			ID:     p.ID,
			Module: p.Module,
			Action: p.Action,
			Scope:  p.Scope,
		})
	}

	return &GetRoleRes{RoleDetail: &RoleDetail{
		ID:          roleWithPerms.ID,
		Name:        roleWithPerms.Name,
		Label:       roleWithPerms.Label,
		Description: roleWithPerms.Description,
		IsSystem:    roleWithPerms.IsSystem,
		Permissions: perms,
		CreatedAt:   roleWithPerms.CreatedAt.Format("2006-01-02 15:04:05"),
	}}, nil
}

// CreateRole 创建角色
func (c *PermissionController) CreateRole(ctx context.Context, req *CreateRoleReq) (res *CreateRoleRes, err error) {
	perms := make([]model.Permission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, model.Permission{
			Module: p.Module,
			Action: p.Action,
			Scope:  p.Scope,
		})
	}

	role, err := c.logic.CreateRole(req.Name, req.Label, req.Description, perms)
	if err != nil {
		return nil, err
	}
	return &CreateRoleRes{ID: role.ID}, nil
}

// UpdateRole 编辑角色
func (c *PermissionController) UpdateRole(ctx context.Context, req *UpdateRoleReq) (res *UpdateRoleRes, err error) {
	perms := make([]model.Permission, 0, len(req.Permissions))
	for _, p := range req.Permissions {
		perms = append(perms, model.Permission{
			Module: p.Module,
			Action: p.Action,
			Scope:  p.Scope,
		})
	}

	if err := c.logic.UpdateRole(req.ID, req.Label, req.Description, perms); err != nil {
		return nil, err
	}
	return &UpdateRoleRes{}, nil
}

// DeleteRole 删除角色
func (c *PermissionController) DeleteRole(ctx context.Context, req *DeleteRoleReq) (res *DeleteRoleRes, err error) {
	if err := c.logic.DeleteRole(req.ID); err != nil {
		return nil, err
	}
	return &DeleteRoleRes{}, nil
}

// AvailablePermissions 获取所有可用权限
func (c *PermissionController) AvailablePermissions(ctx context.Context, req *AvailablePermissionsReq) (res *AvailablePermissionsRes, err error) {
	groups := c.logic.GetAllAvailablePermissions()

	groupDetails := make([]PermissionGroupDetail, 0, len(groups))
	for _, g := range groups {
		permDetails := make([]PermissionDefDetail, 0, len(g.Permissions))
		for _, p := range g.Permissions {
			permDetails = append(permDetails, PermissionDefDetail{
				Action:      p.Action,
				Description: p.Description,
				Scopes:      p.Scopes,
			})
		}
		groupDetails = append(groupDetails, PermissionGroupDetail{
			Module:      g.Module,
			Permissions: permDetails,
		})
	}

	return &AvailablePermissionsRes{Groups: groupDetails}, nil
}

// AssignRoles 为用户分配角色
func (c *PermissionController) AssignRoles(ctx context.Context, req *AssignRolesReq) (res *AssignRolesRes, err error) {
	if err := c.logic.AssignRolesToUser(req.UserID, req.RoleIDs); err != nil {
		return nil, err
	}
	return &AssignRolesRes{}, nil
}

// GetUserRoles 获取用户角色
func (c *PermissionController) GetUserRoles(ctx context.Context, req *GetUserRolesReq) (res *GetUserRolesRes, err error) {
	roles, err := c.logic.GetUserRoles(req.UserID)
	if err != nil {
		return nil, err
	}

	roleSimples := make([]RoleSimple, 0, len(roles))
	for _, r := range roles {
		roleSimples = append(roleSimples, RoleSimple{
			ID:    r.ID,
			Name:  r.Name,
			Label: r.Label,
		})
	}

	return &GetUserRolesRes{Roles: roleSimples}, nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

// GetCurrentUserID 从 context 中获取当前用户 ID
func GetCurrentUserID(ctx context.Context) int64 {
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return 0
	}
	return r.GetCtxVar("user_id").Int64()
}
