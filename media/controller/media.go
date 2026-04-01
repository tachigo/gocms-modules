// Package controller media API 控制器
// 媒体文件管理（/api/admin/media）+ 文件夹管理（/api/admin/media/folders）
package controller

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"

	"gocms/module/media/logic"
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
// Request / Response
// ---------------------------------------------------------------------------

// --- 媒体列表 ---
type ListMediaReq struct {
	g.Meta       `path:"/media" method:"GET" tags:"媒体管理" summary:"媒体列表" dc:"分页获取媒体文件"`
	FolderID     *int64 `json:"folder_id" dc:"文件夹ID"`
	MimePrefix   string `json:"mime_prefix" dc:"MIME类型前缀过滤（如 image）"`
	Page         int    `json:"page" d:"1" v:"min:1" dc:"页码"`
	PageSize     int    `json:"page_size" d:"20" v:"min:1|max:100" dc:"每页条数"`
}
type ListMediaRes struct {
	g.Meta   `mime:"application/json"`
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

// --- 上传文件（手动处理 multipart） ---
type UploadMediaReq struct {
	g.Meta `path:"/media/upload" method:"POST" mime:"multipart/form-data" tags:"媒体管理" summary:"上传文件" dc:"上传单个文件"`
	FolderID *int64 `json:"folder_id" dc:"文件夹ID"`
}
type UploadMediaRes struct {
	g.Meta   `mime:"application/json"`
	ID       int64  `json:"id"`
	URL      string `json:"url"`
	Filename string `json:"filename"`
	MimeType string `json:"mime_type"`
	Size     int64  `json:"size"`
}

// --- 媒体详情 ---
type GetMediaReq struct {
	g.Meta `path:"/media/{id}" method:"GET" tags:"媒体管理" summary:"媒体详情"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"媒体ID"`
}
type GetMediaRes struct {
	g.Meta `mime:"application/json"`
	*MediaDetail
}
type MediaDetail struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	URL         string `json:"url"`
	MimeType    string `json:"mime_type"`
	Size        int64  `json:"size"`
	Alt         string `json:"alt"`
	Title       string `json:"title"`
	UploadedBy  int64  `json:"uploaded_by"`
	CreatedAt   string `json:"created_at"`
}

// --- 更新元信息 ---
type UpdateMediaReq struct {
	g.Meta `path:"/media/{id}" method:"PUT" tags:"媒体管理" summary:"更新元信息" dc:"更新 Alt/Title"`
	ID     int64  `json:"id" in:"path" v:"required|min:1" dc:"媒体ID"`
	Alt    string `json:"alt" dc:"Alt文本"`
	Title  string `json:"title" dc:"标题"`
}
type UpdateMediaRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除文件 ---
type DeleteMediaReq struct {
	g.Meta `path:"/media/{id}" method:"DELETE" tags:"媒体管理" summary:"删除文件"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"媒体ID"`
}
type DeleteMediaRes struct {
	g.Meta `mime:"application/json"`
}

// --- 文件夹树 ---
type ListFoldersReq struct {
	g.Meta `path:"/media/folders" method:"GET" tags:"媒体管理" summary:"文件夹树"`
}
type ListFoldersRes struct {
	g.Meta `mime:"application/json"`
	List   interface{} `json:"list"`
}

// --- 创建文件夹 ---
type CreateFolderReq struct {
	g.Meta   `path:"/media/folders" method:"POST" tags:"媒体管理" summary:"创建文件夹"`
	Name     string `json:"name" v:"required" dc:"文件夹名称"`
	ParentID *int64 `json:"parent_id" dc:"父级文件夹ID"`
}
type CreateFolderRes struct {
	g.Meta `mime:"application/json"`
	ID     int64 `json:"id"`
}

// --- 重命名文件夹 ---
type RenameFolderReq struct {
	g.Meta `path:"/media/folders/{id}" method:"PUT" tags:"媒体管理" summary:"重命名文件夹"`
	ID     int64  `json:"id" in:"path" v:"required|min:1" dc:"文件夹ID"`
	Name   string `json:"name" v:"required" dc:"新名称"`
}
type RenameFolderRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除文件夹 ---
type DeleteFolderReq struct {
	g.Meta `path:"/media/folders/{id}" method:"DELETE" tags:"媒体管理" summary:"删除文件夹" dc:"仅空文件夹可删除"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"文件夹ID"`
}
type DeleteFolderRes struct {
	g.Meta `mime:"application/json"`
}

// ---------------------------------------------------------------------------
// Controller
// ---------------------------------------------------------------------------

// AdminController 媒体管理控制器
type AdminController struct {
	logic *logic.Logic
}

// NewAdminController 创建媒体管理控制器
func NewAdminController(l *logic.Logic) *AdminController {
	return &AdminController{logic: l}
}

// ListMedia 媒体列表
func (c *AdminController) ListMedia(ctx context.Context, req *ListMediaReq) (res *ListMediaRes, err error) {
	ownerID := rbacOwnerFilter(ctx)
	items, total, err := c.logic.List(req.FolderID, req.MimePrefix, req.Page, req.PageSize, ownerID)
	if err != nil {
		return nil, err
	}
	return &ListMediaRes{List: items, Total: total, Page: req.Page, PageSize: req.PageSize}, nil
}

// UploadMedia 上传文件
func (c *AdminController) UploadMedia(ctx context.Context, req *UploadMediaReq) (res *UploadMediaRes, err error) {
	r := ghttp.RequestFromCtx(ctx)
	file, header, err := r.Request.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("请选择要上传的文件")
	}
	defer file.Close()

	userID := r.GetCtxVar("user_id").Int64()
	media, err := c.logic.Upload(file, header, req.FolderID, userID)
	if err != nil {
		return nil, err
	}
	return &UploadMediaRes{ID: media.ID, URL: media.URL, Filename: media.Filename, MimeType: media.MimeType, Size: media.Size}, nil
}

// GetMedia 媒体详情
func (c *AdminController) GetMedia(ctx context.Context, req *GetMediaReq) (res *GetMediaRes, err error) {
	media, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, media.UploadedBy)
	return &GetMediaRes{MediaDetail: &MediaDetail{
		ID:         media.ID,
		Filename:   media.Filename,
		URL:        media.URL,
		MimeType:   media.MimeType,
		Size:       media.Size,
		Alt:        media.Alt,
		Title:      media.Title,
		UploadedBy: media.UploadedBy,
		CreatedAt:  media.CreatedAt.Format("2006-01-02 15:04:05"),
	}}, nil
}

// UpdateMedia 更新元信息
func (c *AdminController) UpdateMedia(ctx context.Context, req *UpdateMediaReq) (res *UpdateMediaRes, err error) {
	// RBAC scope:own 检查 — 确认当前用户有权操作该媒体文件
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.UploadedBy)

	if err := c.logic.Update(req.ID, req.Alt, req.Title); err != nil {
		return nil, err
	}
	return &UpdateMediaRes{}, nil
}

// DeleteMedia 删除文件
func (c *AdminController) DeleteMedia(ctx context.Context, req *DeleteMediaReq) (res *DeleteMediaRes, err error) {
	// RBAC scope:own 检查 — 确认当前用户有权操作该媒体文件
	existing, err := c.logic.GetByID(req.ID)
	if err != nil {
		return nil, err
	}
	enforceRBACScope(ctx, existing.UploadedBy)

	if err := c.logic.Delete(req.ID); err != nil {
		return nil, err
	}
	return &DeleteMediaRes{}, nil
}

// ListFolders 文件夹树
func (c *AdminController) ListFolders(ctx context.Context, req *ListFoldersReq) (res *ListFoldersRes, err error) {
	folders, err := c.logic.ListFolders()
	if err != nil {
		return nil, err
	}
	return &ListFoldersRes{List: folders}, nil
}

// CreateFolder 创建文件夹
func (c *AdminController) CreateFolder(ctx context.Context, req *CreateFolderReq) (res *CreateFolderRes, err error) {
	folder, err := c.logic.CreateFolder(req.Name, req.ParentID)
	if err != nil {
		return nil, err
	}
	return &CreateFolderRes{ID: folder.ID}, nil
}

// RenameFolder 重命名文件夹
func (c *AdminController) RenameFolder(ctx context.Context, req *RenameFolderReq) (res *RenameFolderRes, err error) {
	if err := c.logic.RenameFolder(req.ID, req.Name); err != nil {
		return nil, err
	}
	return &RenameFolderRes{}, nil
}

// DeleteFolder 删除文件夹
func (c *AdminController) DeleteFolder(ctx context.Context, req *DeleteFolderReq) (res *DeleteFolderRes, err error) {
	if err := c.logic.DeleteFolder(req.ID); err != nil {
		return nil, err
	}
	return &DeleteFolderRes{}, nil
}
