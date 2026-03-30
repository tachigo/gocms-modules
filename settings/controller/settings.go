// Package controller settings API 控制器
// GoFrame Bind 模式：Request/Response struct + Controller 方法
package controller

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"gocms/internal/module/settings/logic"
	"gocms/internal/module/settings/model"
)

// ---------------------------------------------------------------------------
// Request / Response 定义（GoFrame Bind 自动生成 OpenAPI）
// ---------------------------------------------------------------------------

// GetPublicSettingsReq 获取公开站点配置
type GetPublicSettingsReq struct {
	g.Meta `path:"/settings" method:"GET" tags:"站点配置" summary:"获取站点配置（公开）" dc:"返回站点基本配置，不含敏感信息"`
}

// GetPublicSettingsRes 公开配置响应
type GetPublicSettingsRes struct {
	g.Meta `mime:"application/json"`
	*model.PublicConfig
}

// GetAdminSettingsReq 获取完整站点配置（管理员）
type GetAdminSettingsReq struct {
	g.Meta `path:"/settings" method:"GET" tags:"站点配置" summary:"获取完整站点配置" dc:"返回完整配置，含敏感信息，需管理员权限"`
}

// GetAdminSettingsRes 完整配置响应
type GetAdminSettingsRes struct {
	g.Meta `mime:"application/json"`
	*model.SiteConfig
}

// ---------------------------------------------------------------------------
// Controller
// ---------------------------------------------------------------------------

// PublicController 公开 API 控制器（/api/settings）
type PublicController struct {
	logic *logic.Logic
}

// NewPublicController 创建公开控制器
func NewPublicController(l *logic.Logic) *PublicController {
	return &PublicController{logic: l}
}

// GetPublicSettings 获取公开站点配置
func (c *PublicController) GetPublicSettings(ctx context.Context, req *GetPublicSettingsReq) (res *GetPublicSettingsRes, err error) {
	config := c.logic.GetPublicConfig()
	return &GetPublicSettingsRes{PublicConfig: config}, nil
}

// AdminController 管理 API 控制器（/api/admin/settings）
type AdminController struct {
	logic *logic.Logic
}

// NewAdminController 创建管理控制器
func NewAdminController(l *logic.Logic) *AdminController {
	return &AdminController{logic: l}
}

// GetAdminSettings 获取完整站点配置
func (c *AdminController) GetAdminSettings(ctx context.Context, req *GetAdminSettingsReq) (res *GetAdminSettingsRes, err error) {
	config := c.logic.GetFullConfig()
	return &GetAdminSettingsRes{SiteConfig: config}, nil
}
