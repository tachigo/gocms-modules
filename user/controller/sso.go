// Package controller user 模块控制器
// SSO 测试接口 - 用于验证 Slave 模式下用户信息透传
package controller

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"gocms/core"
)

// ---------------------------------------------------------------------------
// SSO 测试接口
// ---------------------------------------------------------------------------

type SSOTestReq struct {
	g.Meta `path:"/admin/sso-test" method:"GET" tags:"SSO测试" summary:"SSO用户信息回显" dc:"返回当前请求上下文中的SSO用户信息，用于验证Slave模式配置"`
}

type SSOTestRes struct {
	g.Meta      `mime:"application/json"`
	UserInfo    *core.UserInfo    `json:"user_info" dc:"从Context提取的用户信息"`
	RawHeaders  map[string]string `json:"raw_headers" dc:"原始请求头（SSO相关）"`
	Mode        string            `json:"mode" dc:"当前运行模式"`
}

// SSOTestController SSO测试控制器
type SSOTestController struct{}

// NewSSOTestController 创建SSO测试控制器
func NewSSOTestController() *SSOTestController {
	return &SSOTestController{}
}

// SSOTest 回显SSO用户信息
// 用于验证：
// 1. SSOMiddleware 是否正确解析了 Header
// 2. UserInfo 是否正确注入 Context
// 3. 多角色格式是否正确处理
func (c *SSOTestController) SSOTest(ctx context.Context, req *SSOTestReq) (*SSOTestRes, error) {
	// 从 Context 提取用户信息
	userInfo := core.GetUserFromCtx(ctx)

	// 获取原始请求对象以读取 headers
	r := g.RequestFromCtx(ctx)

	// 收集原始 SSO Header
	rawHeaders := map[string]string{
		"X-SSO-User-ID":    r.GetHeader("X-SSO-User-ID"),
		"X-SSO-User-Name":  r.GetHeader("X-SSO-User-Name"),
		"X-SSO-User-Email": r.GetHeader("X-SSO-User-Email"),
		"X-SSO-User-Role":  r.GetHeader("X-SSO-User-Role"),
		"Authorization":    r.GetHeader("Authorization"),
	}

	return &SSOTestRes{
		UserInfo:   userInfo,
		RawHeaders: rawHeaders,
		Mode:       "slave",
	}, nil
}