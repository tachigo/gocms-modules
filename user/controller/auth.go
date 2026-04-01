// Package controller user 认证 API 控制器
// 公开 API（login）+ 需认证 API（logout/profile/password）
package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/core"
	"gocms/module/user/logic"
)

// ---------------------------------------------------------------------------
// Request / Response 定义
// ---------------------------------------------------------------------------

// --- 登录（公开） ---

type LoginReq struct {
	g.Meta   `path:"/auth/login" method:"POST" tags:"认证" summary:"用户登录" dc:"用户名密码登录，返回 JWT Token"`
	Username string `json:"username" v:"required" dc:"用户名"`
	Password string `json:"password" v:"required" dc:"密码"`
}

type LoginRes struct {
	g.Meta `mime:"application/json"`
	Token  string      `json:"token" dc:"JWT Token"`
	User   interface{} `json:"user" dc:"用户信息"`
}

// --- 登出（需认证） ---

type LogoutReq struct {
	g.Meta `path:"/auth/logout" method:"POST" tags:"认证" summary:"用户登出" dc:"将当前 Token 加入黑名单"`
}

type LogoutRes struct {
	g.Meta `mime:"application/json"`
}

// --- 获取个人信息（需认证） ---

type GetProfileReq struct {
	g.Meta `path:"/auth/profile" method:"GET" tags:"认证" summary:"获取个人信息" dc:"获取当前登录用户的个人信息"`
}

type GetProfileRes struct {
	g.Meta `mime:"application/json"`
	*ProfileData
}

type ProfileData struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// --- 更新个人信息（需认证） ---

type UpdateProfileReq struct {
	g.Meta   `path:"/auth/profile" method:"PUT" tags:"认证" summary:"更新个人信息" dc:"更新昵称、头像等"`
	Nickname string `json:"nickname" dc:"昵称"`
	Avatar   string `json:"avatar" dc:"头像 URL"`
}

type UpdateProfileRes struct {
	g.Meta `mime:"application/json"`
}

// --- 修改密码（需认证） ---

type ChangePasswordReq struct {
	g.Meta      `path:"/auth/password" method:"PUT" tags:"认证" summary:"修改密码" dc:"验证旧密码后设置新密码"`
	OldPassword string `json:"old_password" v:"required" dc:"旧密码"`
	NewPassword string `json:"new_password" v:"required|min-length:6" dc:"新密码（最少6位）"`
}

type ChangePasswordRes struct {
	g.Meta `mime:"application/json"`
}

// ---------------------------------------------------------------------------
// Public Controller（公开 API，无需认证）
// ---------------------------------------------------------------------------

// AuthPublicController 公开认证控制器（登录）
type AuthPublicController struct {
	logic *logic.UserLogic
}

// NewAuthPublicController 创建公开认证控制器
func NewAuthPublicController(l *logic.UserLogic) *AuthPublicController {
	return &AuthPublicController{logic: l}
}

// Login 用户登录
func (c *AuthPublicController) Login(ctx context.Context, req *LoginReq) (res *LoginRes, err error) {
	token, user, err := c.logic.Login(req.Username, req.Password)
	if err != nil {
		return nil, err
	}
	return &LoginRes{
		Token: token,
		User: &ProfileData{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Nickname: user.Nickname,
			Avatar:   user.Avatar,
		},
	}, nil
}

// ---------------------------------------------------------------------------
// Protected Controller（需认证 API）
// ---------------------------------------------------------------------------

// AuthProtectedController 需认证的认证控制器（登出/个人信息/改密码）
type AuthProtectedController struct {
	logic *logic.UserLogic
}

// NewAuthProtectedController 创建需认证的认证控制器
func NewAuthProtectedController(l *logic.UserLogic) *AuthProtectedController {
	return &AuthProtectedController{logic: l}
}

// Logout 用户登出
func (c *AuthProtectedController) Logout(ctx context.Context, req *LogoutReq) (res *LogoutRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	// 从 Authorization header 获取 token
	token := extractToken(r)
	if token != "" {
		c.logic.Logout(token)
	}
	return &LogoutRes{}, nil
}

// GetProfile 获取个人信息
// slave 模式下直接通过 core.GetUserFromCtx(ctx) 获取上下文中的用户信息，不查询数据库
func (c *AuthProtectedController) GetProfile(ctx context.Context, req *GetProfileReq) (res *GetProfileRes, err error) {
	// 优先从 Context 获取用户信息（支持 slave 模式）
	userInfo := core.GetUserFromCtx(ctx)
	if userInfo != nil {
		// slave 模式：直接返回 Context 中的用户信息
		return &GetProfileRes{ProfileData: &ProfileData{
			ID:       userInfo.ID,
			Username: userInfo.Username,
			Email:    userInfo.Email,
			// slave 模式下 Nickname 和 Avatar 可能为空
		}}, nil
	}

	// master 模式：从数据库查询
	userID := GetCurrentUserID(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("未登录")
	}

	user, err := c.logic.GetProfile(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &GetProfileRes{ProfileData: &ProfileData{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Nickname: user.Nickname,
		Avatar:   user.Avatar,
	}}, nil
}

// UpdateProfile 更新个人信息
func (c *AuthProtectedController) UpdateProfile(ctx context.Context, req *UpdateProfileReq) (res *UpdateProfileRes, err error) {
	userID := GetCurrentUserID(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("未登录")
	}

	if err := c.logic.UpdateProfile(userID, req.Nickname, req.Avatar); err != nil {
		return nil, err
	}
	return &UpdateProfileRes{}, nil
}

// ChangePassword 修改密码
func (c *AuthProtectedController) ChangePassword(ctx context.Context, req *ChangePasswordReq) (res *ChangePasswordRes, err error) {
	userID := GetCurrentUserID(ctx)
	if userID == 0 {
		return nil, fmt.Errorf("未登录")
	}

	if err := c.logic.ChangePassword(userID, req.OldPassword, req.NewPassword); err != nil {
		return nil, err
	}
	return &ChangePasswordRes{}, nil
}

// ---------------------------------------------------------------------------
// 辅助函数
// ---------------------------------------------------------------------------

// GetCurrentUserID 从 context 中获取当前用户 ID（JWT 中间件注入）
func GetCurrentUserID(ctx context.Context) int64 {
	r := ghttp.RequestFromCtx(ctx)
	if r == nil {
		return 0
	}
	return r.GetCtxVar("user_id").Int64()
}

// extractToken 从请求 header 中提取 JWT Token
func extractToken(r *ghttp.Request) string {
	auth := r.GetHeader("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return auth[7:]
	}
	return ""
}
