// Package controller article API 控制器
// 公开 API（/api/articles） + 管理 API（/api/admin/articles）
package controller

import (
	"context"
	"net/http"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/internal/module/article/logic"
	"gocms/internal/module/article/model"
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
// scope=all 或未设置时返回 0（不过滤）
func rbacOwnerFilter(ctx context.Context) int64 {
	r := ghttp.RequestFromCtx(ctx)
	if r.GetCtxVar("rbac_scope").String() == "own" {
		return r.GetCtxVar("rbac_user_id").Int64()
	}
	return 0
}

// ---------------------------------------------------------------------------
// 公开 API Request / Response 定义
// ---------------------------------------------------------------------------

// --- 公开文章列表 ---
type ListPublicArticlesReq struct {
	g.Meta   `path:"/articles" method:"GET" tags:"文章" summary:"文章列表" dc:"获取已发布文章列表（公开API）"`
	Page     int `json:"page" d:"1" v:"min:1" dc:"页码"`
	PageSize int `json:"page_size" d:"20" v:"min:1|max:100" dc:"每页条数"`
}
type ListPublicArticlesRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"文章列表"`
	Total    int64       `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"page_size" dc:"每页条数"`
}

// --- 公开文章详情（ID） ---
type GetPublicArticleReq struct {
	g.Meta `path:"/articles/{id}" method:"GET" tags:"文章" summary:"文章详情" dc:"通过ID获取已发布文章"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"文章ID"`
}
type GetPublicArticleRes struct {
	g.Meta `mime:"application/json"`
	*ArticleDetail
}

// --- 公开文章详情（Slug） ---
type GetPublicArticleBySlugReq struct {
	g.Meta `path:"/articles/slug/{slug}" method:"GET" tags:"文章" summary:"文章详情（Slug）" dc:"通过别名获取已发布文章"`
	Slug   string `json:"slug" in:"path" v:"required" dc:"文章别名"`
}
type GetPublicArticleBySlugRes struct {
	g.Meta `mime:"application/json"`
	*ArticleDetail
}

// ArticleDetail 文章详情响应
type ArticleDetail struct {
	ID          int64    `json:"id"`
	Title       string   `json:"title"`
	Slug        string   `json:"slug"`
	Summary     string   `json:"summary"`
	Body        string   `json:"body"`
	CoverImage  *int64   `json:"cover_image"`
	AuthorID    int64    `json:"author_id"`
	Status      string   `json:"status"`
	PublishedAt string   `json:"published_at"`
	IsTop       bool     `json:"is_top"`
	SeoTitle    string   `json:"seo_title"`
	SeoDesc     string   `json:"seo_desc"`
	CategoryIDs []int64  `json:"category_ids"`
	TagIDs      []int64  `json:"tag_ids"`
	CreatedAt   string   `json:"created_at"`
	UpdatedAt   string   `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// 管理 API Request / Response 定义
// ---------------------------------------------------------------------------

// --- 管理文章列表 ---
type ListAdminArticlesReq struct {
	g.Meta   `path:"/articles" method:"GET" tags:"文章管理" summary:"文章列表" dc:"获取文章管理列表（全部状态）"`
	Status   string `json:"status" dc:"状态筛选：draft/published/archived"`
	Page     int    `json:"page" d:"1" v:"min:1" dc:"页码"`
	PageSize int    `json:"page_size" d:"20" v:"min:1|max:100" dc:"每页条数"`
}
type ListAdminArticlesRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list" dc:"文章列表"`
	Total    int64       `json:"total" dc:"总数"`
	Page     int         `json:"page" dc:"当前页"`
	PageSize int         `json:"page_size" dc:"每页条数"`
}

// --- 创建文章 ---
type CreateArticleReq struct {
	g.Meta      `path:"/articles" method:"POST" tags:"文章管理" summary:"创建文章" dc:"创建新文章"`
	Title       string  `json:"title" v:"required|max-length:200" dc:"文章标题"`
	Slug        string  `json:"slug" v:"required|max-length:200|regex:^[a-z0-9-]+$" dc:"URL别名（小写字母、数字、连字符）"`
	Summary     string  `json:"summary" dc:"文章摘要"`
	Body        string  `json:"body" v:"required" dc:"文章正文（HTML）"`
	CoverImage  *int64  `json:"cover_image" dc:"封面图ID（关联media）"`
	AuthorID    int64   `json:"author_id" dc:"作者ID（自动从JWT填充，无需前端传）"`
	Status      string  `json:"status" v:"in:draft,published,archived" d:"draft" dc:"状态：draft/published/archived"`
	IsTop       bool    `json:"is_top" d:"false" dc:"是否置顶"`
	SeoTitle    string  `json:"seo_title" dc:"SEO标题"`
	SeoDesc     string  `json:"seo_desc" dc:"SEO描述"`
	CategoryIDs []int64 `json:"category_ids" dc:"分类ID列表"`
	TagIDs      []int64 `json:"tag_ids" dc:"标签ID列表"`
}
type CreateArticleRes struct {
	g.Meta `mime:"application/json"`
	ID     int64 `json:"id" dc:"文章ID"`
}

// --- 文章详情（管理） ---
type GetAdminArticleReq struct {
	g.Meta `path:"/articles/{id}" method:"GET" tags:"文章管理" summary:"文章详情" dc:"获取文章管理详情"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"文章ID"`
}
type GetAdminArticleRes struct {
	g.Meta `mime:"application/json"`
	*ArticleDetail
}

// --- 更新文章 ---
type UpdateArticleReq struct {
	g.Meta      `path:"/articles/{id}" method:"PUT" tags:"文章管理" summary:"更新文章" dc:"更新文章内容"`
	ID          int64   `json:"id" in:"path" v:"required|min:1" dc:"文章ID"`
	Title       string  `json:"title" dc:"文章标题"`
	Slug        string  `json:"slug" v:"max-length:200|regex:^[a-z0-9-]*$" dc:"URL别名"`
	Summary     string  `json:"summary" dc:"文章摘要"`
	Body        string  `json:"body" dc:"文章正文"`
	CoverImage  *int64  `json:"cover_image" dc:"封面图ID"`
	AuthorID    int64   `json:"author_id" dc:"作者ID"`
	Status      string  `json:"status" v:"in:draft,published,archived" dc:"状态"`
	IsTop       *bool   `json:"is_top" dc:"是否置顶"`
	SeoTitle    string  `json:"seo_title" dc:"SEO标题"`
	SeoDesc     string  `json:"seo_desc" dc:"SEO描述"`
	CategoryIDs []int64 `json:"category_ids" dc:"分类ID列表"`
	TagIDs      []int64 `json:"tag_ids" dc:"标签ID列表"`
}
type UpdateArticleRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除文章 ---
type DeleteArticleReq struct {
	g.Meta `path:"/articles/{id}" method:"DELETE" tags:"文章管理" summary:"删除文章" dc:"软删除文章"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"文章ID"`
}
type DeleteArticleRes struct {
	g.Meta `mime:"application/json"`
}

// --- 发布文章 ---
type PublishArticleReq struct {
	g.Meta `path:"/articles/{id}/publish" method:"POST" tags:"文章管理" summary:"发布文章" dc:"将文章状态设为published"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"文章ID"`
}
type PublishArticleRes struct {
	g.Meta `mime:"application/json"`
}

// --- 取消发布文章 ---
type UnpublishArticleReq struct {
	g.Meta `path:"/articles/{id}/unpublish" method:"POST" tags:"文章管理" summary:"取消发布" dc:"将文章状态设为draft"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"文章ID"`
}
type UnpublishArticleRes struct {
	g.Meta `mime:"application/json"`
}

// ---------------------------------------------------------------------------
// 控制器实现
// ---------------------------------------------------------------------------

// PublicController 公开文章控制器（/api/articles）
type PublicController struct {
	logic *logic.Logic
}

// NewPublicController 创建公开文章控制器
func NewPublicController(l *logic.Logic) *PublicController {
	return &PublicController{logic: l}
}

// ListPublicArticles 公开文章列表
func (c *PublicController) ListPublicArticles(ctx context.Context, req *ListPublicArticlesReq) (res *ListPublicArticlesRes, err error) {
	articles, total, err := c.logic.ListPublic(req.Page, req.PageSize)
	if err != nil {
		return nil, err
	}
	return &ListPublicArticlesRes{
		List:     articles,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// GetPublicArticle 公开文章详情（ID）
func (c *PublicController) GetPublicArticle(ctx context.Context, req *GetPublicArticleReq) (res *GetPublicArticleRes, err error) {
	article, err := c.logic.GetPublicByID(req.ID)
	if err != nil {
		return nil, err
	}
	return &GetPublicArticleRes{ArticleDetail: convertToDetail(article)}, nil
}

// GetPublicArticleBySlug 公开文章详情（Slug）
func (c *PublicController) GetPublicArticleBySlug(ctx context.Context, req *GetPublicArticleBySlugReq) (res *GetPublicArticleBySlugRes, err error) {
	article, err := c.logic.GetPublicBySlug(req.Slug)
	if err != nil {
		return nil, err
	}
	return &GetPublicArticleBySlugRes{ArticleDetail: convertToDetail(article)}, nil
}

// AdminController 文章管理控制器（/api/admin/articles）
type AdminController struct {
	logic *logic.Logic
}

// NewAdminController 创建文章管理控制器
func NewAdminController(l *logic.Logic) *AdminController {
	return &AdminController{logic: l}
}

// ListAdminArticles 文章管理列表
func (c *AdminController) ListAdminArticles(ctx context.Context, req *ListAdminArticlesReq) (res *ListAdminArticlesRes, err error) {
	ownerID := rbacOwnerFilter(ctx)
	articles, total, err := c.logic.List(req.Status, req.Page, req.PageSize, ownerID)
	if err != nil {
		return nil, err
	}
	return &ListAdminArticlesRes{
		List:     articles,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// CreateArticle 创建文章
func (c *AdminController) CreateArticle(ctx context.Context, req *CreateArticleReq) (res *CreateArticleRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	article := c.reqToModel(req)
	article.AuthorID = userID // 从 JWT token 自动填充 author_id
	article.CreatedBy = userID

	result, err := c.logic.Create(article, req.CategoryIDs, req.TagIDs)
	if err != nil {
		return nil, err
	}
	return &CreateArticleRes{ID: result.ID}, nil
}

// GetAdminArticle 文章详情（管理）
func (c *AdminController) GetAdminArticle(ctx context.Context, req *GetAdminArticleReq) (res *GetAdminArticleRes, err error) {
	article, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, article.AuthorID)
	return &GetAdminArticleRes{ArticleDetail: convertToDetail(article)}, nil
}

// UpdateArticle 更新文章
func (c *AdminController) UpdateArticle(ctx context.Context, req *UpdateArticleReq) (res *UpdateArticleRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	// RBAC scope:own 检查 — 确认当前用户有权操作该文章
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.AuthorID)

	article := c.updateReqToModel(req)
	article.UpdatedBy = &userID

	_, err = c.logic.Update(req.ID, article, req.CategoryIDs, req.TagIDs)
	if err != nil {
		return nil, err
	}
	return &UpdateArticleRes{}, nil
}

// DeleteArticle 删除文章
func (c *AdminController) DeleteArticle(ctx context.Context, req *DeleteArticleReq) (res *DeleteArticleRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	userID := r.GetCtxVar("user_id").Int64()

	// RBAC scope:own 检查 — 确认当前用户有权操作该文章
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.AuthorID)

	if err := c.logic.Delete(req.ID, userID); err != nil {
		return nil, err
	}
	return &DeleteArticleRes{}, nil
}

// PublishArticle 发布文章
func (c *AdminController) PublishArticle(ctx context.Context, req *PublishArticleReq) (res *PublishArticleRes, err error) {
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
	return &PublishArticleRes{}, nil
}

// UnpublishArticle 取消发布文章
func (c *AdminController) UnpublishArticle(ctx context.Context, req *UnpublishArticleReq) (res *UnpublishArticleRes, err error) {
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
	return &UnpublishArticleRes{}, nil
}

// ---------------------------------------------------------------------------
// 辅助方法
// ---------------------------------------------------------------------------

// reqToModel 将 CreateRequest 转换为 Model
func (c *AdminController) reqToModel(req *CreateArticleReq) *model.Article {
	return &model.Article{
		Title:      req.Title,
		Slug:       req.Slug,
		Summary:    req.Summary,
		Body:       req.Body,
		CoverImage: req.CoverImage,
		AuthorID:   req.AuthorID,
		Status:     req.Status,
		IsTop:      req.IsTop,
		SeoTitle:   req.SeoTitle,
		SeoDesc:    req.SeoDesc,
	}
}

// updateReqToModel 将 UpdateRequest 转换为 Model
func (c *AdminController) updateReqToModel(req *UpdateArticleReq) *model.Article {
	article := &model.Article{
		Title:      req.Title,
		Slug:       req.Slug,
		Summary:    req.Summary,
		Body:       req.Body,
		CoverImage: req.CoverImage,
		AuthorID:   req.AuthorID,
		Status:     req.Status,
		SeoTitle:   req.SeoTitle,
		SeoDesc:    req.SeoDesc,
	}
	if req.IsTop != nil {
		article.IsTop = *req.IsTop
	}
	return article
}

// convertToDetail 将 Model 转换为 Detail
func convertToDetail(article *model.Article) *ArticleDetail {
	if article == nil {
		return nil
	}
	return &ArticleDetail{
		ID:          article.ID,
		Title:       article.Title,
		Slug:        article.Slug,
		Summary:     article.Summary,
		Body:        article.Body,
		CoverImage:  article.CoverImage,
		AuthorID:    article.AuthorID,
		Status:      article.Status,
		PublishedAt: formatTime(article.PublishedAt),
		IsTop:       article.IsTop,
		SeoTitle:    article.SeoTitle,
		SeoDesc:     article.SeoDesc,
		CategoryIDs: article.CategoryIDs,
		TagIDs:      article.TagIDs,
		CreatedAt:   article.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   article.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// formatTime 格式化时间
func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02 15:04:05")
}
