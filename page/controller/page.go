// Package controller page API 控制器
// 页面管理 API（/api/admin/pages）+ 公开 API（/api/pages）
package controller

import (
	"context"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/internal/module/page/logic"
	"gocms/internal/module/page/model"
)

// ---------------------------------------------------------------------------
// RBAC scope:own 辅助函数
// ---------------------------------------------------------------------------

// enforceRBACScope 检查 RBAC scope:own 约束
// 当 scope=own 且资源不属于当前用户时，直接返回 403 并终止请求
func enforceRBACScope(ctx context.Context, resourceOwnerID int64) {
	r := ghttp.RequestFromCtx(ctx)
	if r.GetCtxVar("rbac_scope").String() == "own" {
		if resourceOwnerID != r.GetCtxVar("rbac_user_id").Int64() {
			r.Response.Status = http.StatusForbidden
			r.Response.WriteJsonExit(g.Map{
				"code":    403,
				"message": "没有权限操作他人的资源",
			})
		}
	}
}

// rbacOwnerFilter 返回 scope=own 时的 ownerID，用于 List 查询过滤
func rbacOwnerFilter(ctx context.Context) int64 {
	r := ghttp.RequestFromCtx(ctx)
	if r.GetCtxVar("rbac_scope").String() == "own" {
		return r.GetCtxVar("rbac_user_id").Int64()
	}
	return 0
}

// ---------------------------------------------------------------------------
// Admin Request / Response
// ---------------------------------------------------------------------------

// --- 页面列表 ---
type ListPagesReq struct {
	g.Meta   `path:"/pages" method:"GET" tags:"页面管理" summary:"页面列表" dc:"分页获取页面列表（支持按状态筛选）"`
	Status   string `json:"status" dc:"状态筛选: draft | published"`
	Page     int    `json:"page" d:"1" v:"min:1" dc:"页码"`
	PageSize int    `json:"page_size" d:"20" v:"min:1|max:100" dc:"每页条数"`
}
type ListPagesRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"页面列表"`
	Total    int64       `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"page_size" dc:"每页条数"`
}

// --- 创建页面 ---
type CreatePageReq struct {
	g.Meta        `path:"/pages" method:"POST" tags:"页面管理" summary:"创建页面" dc:"创建新页面（默认为草稿）"`
	Title         string       `json:"title" v:"required|max-length:255" dc:"页面标题"`
	Slug          string       `json:"slug" v:"required|regex:^[a-z0-9-]+$" dc:"URL slug（仅小写字母、数字、连字符）"`
	Body          string       `json:"body" dc:"页面内容（HTML/Markdown）"`
	Excerpt       string       `json:"excerpt" dc:"摘要"`
	FeaturedImage string       `json:"featured_image" dc:"特色图片 URL"`
	Template      string       `json:"template" dc:"页面模板"`
	SortOrder     int          `json:"sort_order" d:"0" dc:"排序权重"`
	PageMeta      model.PageMeta `json:"page_meta" dc:"SEO 元数据"`
	SeoTitle      string       `json:"seo_title" dc:"SEO标题"`
	SeoDesc       string       `json:"seo_desc" dc:"SEO描述"`
	SeoKeywords   string       `json:"seo_keywords" dc:"SEO关键词"`
}
type CreatePageRes struct {
	g.Meta `mime:"application/json"`
	ID     int64 `json:"id" dc:"页面ID"`
}

// --- 页面详情 ---
type GetPageReq struct {
	g.Meta `path:"/pages/{id}" method:"GET" tags:"页面管理" summary:"页面详情" dc:"获取指定页面信息"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"页面ID"`
}
type GetPageRes struct {
	g.Meta `mime:"application/json"`
	*PageDetail
}
type PageDetail struct {
	ID            int64            `json:"id"`
	Title         string           `json:"title"`
	Slug          string           `json:"slug"`
	Body          string           `json:"body"`
	Excerpt       string           `json:"excerpt"`
	Status        model.PageStatus `json:"status"`
	FeaturedImage string           `json:"featured_image"`
	AuthorID      int64            `json:"author_id"`
	Template      string           `json:"template"`
	SortOrder     int              `json:"sort_order"`
	Meta          model.PageMeta   `json:"meta"`
	SeoTitle      string           `json:"seo_title"`
	SeoDesc       string           `json:"seo_desc"`
	SeoKeywords   string           `json:"seo_keywords"`
	PublishedAt   *string          `json:"published_at,omitempty"`
	CreatedAt     string           `json:"created_at"`
	UpdatedAt     string           `json:"updated_at"`
}

// --- 更新页面 ---
type UpdatePageReq struct {
	g.Meta        `path:"/pages/{id}" method:"PUT" tags:"页面管理" summary:"编辑页面" dc:"更新页面信息"`
	ID            int64          `json:"id" in:"path" v:"required|min:1" dc:"页面ID"`
	Title         string         `json:"title" dc:"页面标题"`
	Slug          string         `json:"slug" dc:"URL slug"`
	Body          string         `json:"body" dc:"页面内容"`
	Excerpt       string         `json:"excerpt" dc:"摘要"`
	FeaturedImage string         `json:"featured_image" dc:"特色图片 URL"`
	Template      string         `json:"template" dc:"页面模板"`
	SortOrder     *int            `json:"sort_order,omitempty" dc:"排序权重"`
	PageMeta      *model.PageMeta `json:"page_meta,omitempty" dc:"SEO 元数据"`
	SeoTitle      *string        `json:"seo_title,omitempty" dc:"SEO标题"`
	SeoDesc       *string        `json:"seo_desc,omitempty" dc:"SEO描述"`
	SeoKeywords   *string        `json:"seo_keywords,omitempty" dc:"SEO关键词"`
}
type UpdatePageRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除页面 ---
type DeletePageReq struct {
	g.Meta `path:"/pages/{id}" method:"DELETE" tags:"页面管理" summary:"删除页面" dc:"软删除页面"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"页面ID"`
}
type DeletePageRes struct {
	g.Meta `mime:"application/json"`
}

// --- 发布页面 ---
type PublishPageReq struct {
	g.Meta `path:"/pages/{id}/publish" method:"POST" tags:"页面管理" summary:"发布页面" dc:"将页面状态设为 published"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"页面ID"`
}
type PublishPageRes struct {
	g.Meta `mime:"application/json"`
}

// --- 取消发布 ---
type UnpublishPageReq struct {
	g.Meta `path:"/pages/{id}/unpublish" method:"POST" tags:"页面管理" summary:"取消发布" dc:"将页面状态设回 draft"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"页面ID"`
}
type UnpublishPageRes struct {
	g.Meta `mime:"application/json"`
}

// ---------------------------------------------------------------------------
// Admin Controller
// ---------------------------------------------------------------------------

// AdminController 页面管理控制器（/api/admin/pages）
type AdminController struct {
	logic *logic.Logic
}

// NewAdminController 创建页面管理控制器
func NewAdminController(l *logic.Logic) *AdminController {
	return &AdminController{logic: l}
}

// ListPages 页面列表
func (c *AdminController) ListPages(ctx context.Context, req *ListPagesReq) (res *ListPagesRes, err error) {
	ownerID := rbacOwnerFilter(ctx)
	items, total, err := c.logic.List(req.Status, req.Page, req.PageSize, ownerID)
	if err != nil {
		return nil, err
	}
	return &ListPagesRes{List: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

// CreatePage 创建页面
func (c *AdminController) CreatePage(ctx context.Context, req *CreatePageReq) (res *CreatePageRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	page, err := c.logic.Create(req.Title, req.Slug, req.Body, req.Excerpt, req.FeaturedImage, req.Template, req.SortOrder, req.PageMeta, req.SeoTitle, req.SeoDesc, req.SeoKeywords, userID)
	if err != nil {
		return nil, err
	}
	return &CreatePageRes{ID: page.ID}, nil
}

// GetPage 页面详情
func (c *AdminController) GetPage(ctx context.Context, req *GetPageReq) (res *GetPageRes, err error) {
	page, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, page.AuthorID)
	return &GetPageRes{PageDetail: toPageDetail(page)}, nil
}

// UpdatePage 编辑页面
func (c *AdminController) UpdatePage(ctx context.Context, req *UpdatePageReq) (res *UpdatePageRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	// RBAC scope:own 检查 — 确认当前用户有权操作该页面
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.AuthorID)

	if err := c.logic.Update(req.ID, req.Title, req.Slug, req.Body, req.Excerpt, req.FeaturedImage, req.Template, req.SortOrder, req.PageMeta, req.SeoTitle, req.SeoDesc, req.SeoKeywords, userID); err != nil {
		return nil, err
	}
	return &UpdatePageRes{}, nil
}

// DeletePage 删除页面
func (c *AdminController) DeletePage(ctx context.Context, req *DeletePageReq) (res *DeletePageRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	// RBAC scope:own 检查 — 确认当前用户有权操作该页面
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.AuthorID)

	if err := c.logic.Delete(req.ID, userID); err != nil {
		return nil, err
	}
	return &DeletePageRes{}, nil
}

// PublishPage 发布页面
func (c *AdminController) PublishPage(ctx context.Context, req *PublishPageReq) (res *PublishPageRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	// RBAC scope:own 检查
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.AuthorID)

	if err := c.logic.Publish(req.ID, userID); err != nil {
		return nil, err
	}
	return &PublishPageRes{}, nil
}

// UnpublishPage 取消发布
func (c *AdminController) UnpublishPage(ctx context.Context, req *UnpublishPageReq) (res *UnpublishPageRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	// RBAC scope:own 检查
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.AuthorID)

	if err := c.logic.Unpublish(req.ID, userID); err != nil {
		return nil, err
	}
	return &UnpublishPageRes{}, nil
}

// ---------------------------------------------------------------------------
// Public Request / Response
// ---------------------------------------------------------------------------

// --- 公开页面列表 ---
type PublicListPagesReq struct {
	g.Meta   `path:"/pages" method:"GET" tags:"公开页面" summary:"页面列表" dc:"获取已发布的页面列表"`
	Page     int `json:"page" d:"1" v:"min:1" dc:"页码"`
	PageSize int `json:"page_size" d:"20" v:"min:1|max:100" dc:"每页条数"`
}
type PublicListPagesRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"页面列表"`
	Total    int64       `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"page_size" dc:"每页条数"`
}

// --- 公开页面详情（按ID） ---
type PublicGetPageReq struct {
	g.Meta `path:"/pages/{id}" method:"GET" tags:"公开页面" summary:"页面详情" dc:"按ID获取已发布页面"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"页面ID"`
}
type PublicGetPageRes struct {
	g.Meta `mime:"application/json"`
	*PageDetail
}

// --- 按 slug 获取页面 ---
type PublicGetPageBySlugReq struct {
	g.Meta `path:"/pages/slug/{slug}" method:"GET" tags:"公开页面" summary:"按slug获取页面" dc:"通过slug获取已发布页面详情"`
	Slug   string `json:"slug" in:"path" v:"required" dc:"页面slug"`
}
type PublicGetPageBySlugRes struct {
	g.Meta `mime:"application/json"`
	*PageDetail
}

// ---------------------------------------------------------------------------
// Public Controller
// ---------------------------------------------------------------------------

// PublicController 公开页面控制器（/api/pages）
type PublicController struct {
	logic *logic.Logic
}

// NewPublicController 创建公开页面控制器
func NewPublicController(l *logic.Logic) *PublicController {
	return &PublicController{logic: l}
}

// ListPages 公开页面列表（仅 published）
func (c *PublicController) ListPages(ctx context.Context, req *PublicListPagesReq) (res *PublicListPagesRes, err error) {
	items, total, err := c.logic.ListPublished(req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &PublicListPagesRes{List: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

// GetPage 公开页面详情（按ID）
func (c *PublicController) GetPage(ctx context.Context, req *PublicGetPageReq) (res *PublicGetPageRes, err error) {
	page, err := c.logic.GetPublishedByID(req.ID)
	if err != nil {
		return nil, err
	}
	return &PublicGetPageRes{PageDetail: toPageDetail(page)}, nil
}

// GetPageBySlug 按 slug 获取页面
func (c *PublicController) GetPageBySlug(ctx context.Context, req *PublicGetPageBySlugReq) (res *PublicGetPageBySlugRes, err error) {
	page, err := c.logic.GetPublishedBySlug(req.Slug)
	if err != nil {
		return nil, err
	}
	return &PublicGetPageBySlugRes{PageDetail: toPageDetail(page)}, nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func toPageDetail(page *model.Page) *PageDetail {
	detail := &PageDetail{
		ID:            page.ID,
		Title:         page.Title,
		Slug:          page.Slug,
		Body:          page.Body,
		Excerpt:       page.Excerpt,
		Status:        page.Status,
		FeaturedImage: page.FeaturedImage,
		AuthorID:      page.AuthorID,
		Template:      page.Template,
		SortOrder:     page.SortOrder,
		Meta:          page.Meta,
		SeoTitle:      page.SeoTitle,
		SeoDesc:       page.SeoDesc,
		SeoKeywords:   page.SeoKeywords,
		CreatedAt:     page.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:     page.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
	if page.PublishedAt != nil {
		t := page.PublishedAt.Format("2006-01-02 15:04:05")
		detail.PublishedAt = &t
	}
	return detail
}
