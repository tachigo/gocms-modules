// Package controller user 管理 API 控制器
// 用户 CRUD（管理员权限，/api/admin/users）
package controller

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"gocms/module/user/logic"
)

// ---------------------------------------------------------------------------
// Request / Response 定义
// ---------------------------------------------------------------------------

// --- 用户列表 ---

type ListUsersReq struct {
	g.Meta   `path:"/users" method:"GET" tags:"用户管理" summary:"用户列表" dc:"分页获取用户列表"`
	Page     int `json:"page" d:"1" v:"min:1" dc:"页码"`
	PageSize int `json:"page_size" d:"20" v:"min:1|max:100" dc:"每页条数"`
}

type ListUsersRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"用户列表"`
	Total    int64       `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"page_size" dc:"每页条数"`
}

// --- 创建用户 ---

type CreateUserReq struct {
	g.Meta   `path:"/users" method:"POST" tags:"用户管理" summary:"创建用户" dc:"管理员创建新用户"`
	Username string `json:"username" v:"required|min-length:2|max-length:50" dc:"用户名"`
	Email    string `json:"email" v:"required|email" dc:"邮箱"`
	Password string `json:"password" v:"required|min-length:6" dc:"密码（最少6位）"`
	Nickname string `json:"nickname" dc:"昵称"`
}

type CreateUserRes struct {
	g.Meta `mime:"application/json"`
	ID     int64 `json:"id" dc:"用户ID"`
}

// --- 用户详情 ---

type GetUserReq struct {
	g.Meta `path:"/users/{id}" method:"GET" tags:"用户管理" summary:"用户详情" dc:"获取指定用户信息"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"用户ID"`
}

type GetUserRes struct {
	g.Meta `mime:"application/json"`
	*UserDetail
}

type UserDetail struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// --- 编辑用户 ---

type UpdateUserReq struct {
	g.Meta   `path:"/users/{id}" method:"PUT" tags:"用户管理" summary:"编辑用户" dc:"更新用户信息"`
	ID       int64  `json:"id" in:"path" v:"required|min:1" dc:"用户ID"`
	Username string `json:"username" dc:"用户名"`
	Email    string `json:"email" dc:"邮箱"`
	Nickname string `json:"nickname" dc:"昵称"`
	Status   string `json:"status" v:"in:active,disabled" dc:"状态"`
}

type UpdateUserRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除用户 ---

type DeleteUserReq struct {
	g.Meta `path:"/users/{id}" method:"DELETE" tags:"用户管理" summary:"删除用户" dc:"软删除用户"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"用户ID"`
}

type DeleteUserRes struct {
	g.Meta `mime:"application/json"`
}

// ---------------------------------------------------------------------------
// Admin User Controller
// ---------------------------------------------------------------------------

// UserAdminController 用户管理控制器（/api/admin/users）
type UserAdminController struct {
	logic *logic.UserLogic
}

// NewUserAdminController 创建用户管理控制器
func NewUserAdminController(l *logic.UserLogic) *UserAdminController {
	return &UserAdminController{logic: l}
}

// ListUsers 用户列表
func (c *UserAdminController) ListUsers(ctx context.Context, req *ListUsersReq) (res *ListUsersRes, err error) {
	users, total, err := c.logic.List(req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &ListUsersRes{
		List:     users,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// CreateUser 创建用户
func (c *UserAdminController) CreateUser(ctx context.Context, req *CreateUserReq) (res *CreateUserRes, err error) {
	user, err := c.logic.Create(req.Username, req.Email, req.Password, req.Nickname)
	if err != nil {
		return nil, err
	}
	return &CreateUserRes{ID: user.ID}, nil
}

// GetUser 用户详情
func (c *UserAdminController) GetUser(ctx context.Context, req *GetUserReq) (res *GetUserRes, err error) {
	user, err := c.logic.GetByID(ctx, req.ID)
	if err != nil {
		return nil, err
	}
	return &GetUserRes{UserDetail: &UserDetail{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    user.Status,
		CreatedAt: user.CreatedAt.Format("2006-01-02 15:04:05"),
	}}, nil
}

// UpdateUser 编辑用户
func (c *UserAdminController) UpdateUser(ctx context.Context, req *UpdateUserReq) (res *UpdateUserRes, err error) {
	if err := c.logic.Update(req.ID, req.Username, req.Email, req.Nickname, req.Status); err != nil {
		return nil, err
	}
	return &UpdateUserRes{}, nil
}

// DeleteUser 删除用户
func (c *UserAdminController) DeleteUser(ctx context.Context, req *DeleteUserReq) (res *DeleteUserRes, err error) {
	if err := c.logic.Delete(req.ID); err != nil {
		return nil, err
	}
	return &DeleteUserRes{}, nil
}
