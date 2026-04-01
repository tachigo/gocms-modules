// Package controller taxonomy API 控制器
// GoFrame Bind 模式：Request/Response struct + Controller 方法
// 公开 API：按词汇表获取术语列表/详情
// 管理 API：词汇表列表、术语 CRUD
package controller

import (
	"context"

	"github.com/gogf/gf/v2/frame/g"

	"gocms/module/taxonomy/logic"
)

// ---------------------------------------------------------------------------
// Request / Response 定义 — 公开 API
// ---------------------------------------------------------------------------

// --- 获取词汇表下所有术语（公开） ---

type ListTermsPublicReq struct {
	g.Meta     `path:"/taxonomies/{vocabulary}/terms" method:"GET" tags:"分类管理" summary:"获取术语列表（公开）" dc:"获取指定词汇表下所有术语"`
	Vocabulary string `json:"vocabulary" in:"path" v:"required" dc:"词汇表机器名"`
	Hierarchy  bool   `json:"hierarchy" d:"true" dc:"是否返回树形结构"`
}

type ListTermsPublicRes struct {
	g.Meta `mime:"application/json"`
	List   interface{} `json:"list" dc:"术语列表"`
}

// --- 获取术语详情（公开） ---

type GetTermPublicReq struct {
	g.Meta     `path:"/taxonomies/{vocabulary}/terms/{id}" method:"GET" tags:"分类管理" summary:"术语详情（公开）" dc:"获取指定术语详情"`
	Vocabulary string `json:"vocabulary" in:"path" v:"required" dc:"词汇表机器名"`
	ID         int64  `json:"id" in:"path" v:"required|min:1" dc:"术语ID"`
}

type GetTermPublicRes struct {
	g.Meta `mime:"application/json"`
	*TermDetail
}

// TermDetail 术语详情
type TermDetail struct {
	ID           int64  `json:"id"`
	VocabularyID int64  `json:"vocabulary_id"`
	ParentID     *int64 `json:"parent_id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description"`
	Weight       int    `json:"weight"`
	CreatedAt    string `json:"created_at"`
}

// ---------------------------------------------------------------------------
// Request / Response 定义 — 管理 API
// ---------------------------------------------------------------------------

// --- 词汇表列表 ---

type ListVocabulariesReq struct {
	g.Meta `path:"/taxonomies" method:"GET" tags:"分类管理" summary:"词汇表列表" dc:"获取所有词汇表"`
}

type ListVocabulariesRes struct {
	g.Meta `mime:"application/json"`
	List   interface{} `json:"list" dc:"词汇表列表"`
}

// --- 获取词汇表下术语（管理） ---

type ListTermsAdminReq struct {
	g.Meta    `path:"/taxonomies/{vocabulary}/terms" method:"GET" tags:"分类管理" summary:"术语列表" dc:"获取指定词汇表下所有术语"`
	Vocabulary string `json:"vocabulary" in:"path" v:"required" dc:"词汇表机器名（如 article_category）"`
	Hierarchy bool   `json:"hierarchy" d:"true" dc:"是否返回树形结构"`
}

type ListTermsAdminRes struct {
	g.Meta `mime:"application/json"`
	List   interface{} `json:"list" dc:"术语列表"`
}

// --- 创建术语 ---

type CreateTermReq struct {
	g.Meta      `path:"/taxonomies/{vocabulary}/terms" method:"POST" tags:"分类管理" summary:"创建术语" dc:"在指定词汇表下创建新术语"`
	Vocabulary  string `json:"vocabulary" in:"path" v:"required" dc:"词汇表机器名"`
	ParentID    *int64 `json:"parent_id" dc:"父级术语ID（层级分类用）"`
	Name        string `json:"name" v:"required|min-length:1|max-length:100" dc:"术语名称"`
	Slug        string `json:"slug" v:"required|min-length:1|max-length:100" dc:"URL 标识"`
	Description string `json:"description" dc:"描述"`
	Weight      int    `json:"weight" d:"0" dc:"排序权重"`
}

type CreateTermRes struct {
	g.Meta `mime:"application/json"`
	ID     int64 `json:"id" dc:"术语ID"`
}

// --- 更新术语 ---

type UpdateTermReq struct {
	g.Meta      `path:"/taxonomies/terms/{id}" method:"PUT" tags:"分类管理" summary:"更新术语" dc:"更新指定术语"`
	ID          int64  `json:"id" in:"path" v:"required|min:1" dc:"术语ID"`
	ParentID    *int64 `json:"parent_id" dc:"父级术语ID"`
	Name        string `json:"name" dc:"术语名称"`
	Slug        string `json:"slug" dc:"URL 标识"`
	Description string `json:"description" dc:"描述"`
	Weight      int    `json:"weight" d:"0" dc:"排序权重"`
}

type UpdateTermRes struct {
	g.Meta `mime:"application/json"`
}

// --- 删除术语 ---

type DeleteTermReq struct {
	g.Meta `path:"/taxonomies/terms/{id}" method:"DELETE" tags:"分类管理" summary:"删除术语" dc:"删除指定术语（有子术语时不可删除）"`
	ID     int64 `json:"id" in:"path" v:"required|min:1" dc:"术语ID"`
}

type DeleteTermRes struct {
	g.Meta `mime:"application/json"`
}

// ---------------------------------------------------------------------------
// Public Controller — 公开 API
// ---------------------------------------------------------------------------

// PublicController 公开 API 控制器（/api/taxonomies）
type PublicController struct {
	logic *logic.Logic
}

// NewPublicController 创建公开控制器
func NewPublicController(l *logic.Logic) *PublicController {
	return &PublicController{logic: l}
}

// ListTermsPublic 获取词汇表下所有术语（公开）
func (c *PublicController) ListTermsPublic(ctx context.Context, req *ListTermsPublicReq) (res *ListTermsPublicRes, err error) {
	terms, err := c.logic.GetTermsByVocabularyMachineID(req.Vocabulary, req.Hierarchy)
	if err != nil {
		return nil, err
	}
	return &ListTermsPublicRes{List: terms}, nil
}

// GetTermPublic 获取术语详情（公开）
func (c *PublicController) GetTermPublic(ctx context.Context, req *GetTermPublicReq) (res *GetTermPublicRes, err error) {
	term, err := c.logic.GetTermByVocabularyAndID(req.Vocabulary, req.ID)
	if err != nil {
		return nil, err
	}
	return &GetTermPublicRes{TermDetail: &TermDetail{
		ID:           term.ID,
		VocabularyID: term.VocabularyID,
		ParentID:     term.ParentID,
		Name:         term.Name,
		Slug:         term.Slug,
		Description:  term.Description,
		Weight:       term.Weight,
		CreatedAt:    term.CreatedAt.Format("2006-01-02 15:04:05"),
	}}, nil
}

// ---------------------------------------------------------------------------
// Admin Controller — 管理 API
// ---------------------------------------------------------------------------

// AdminController 管理 API 控制器（/api/admin/taxonomies）
type AdminController struct {
	logic *logic.Logic
}

// NewAdminController 创建管理控制器
func NewAdminController(l *logic.Logic) *AdminController {
	return &AdminController{logic: l}
}

// ListVocabularies 获取所有词汇表
func (c *AdminController) ListVocabularies(ctx context.Context, req *ListVocabulariesReq) (res *ListVocabulariesRes, err error) {
	vocabularies, err := c.logic.ListVocabularies()
	if err != nil {
		return nil, err
	}
	return &ListVocabulariesRes{List: vocabularies}, nil
}

// ListTermsAdmin 获取词汇表下术语列表
func (c *AdminController) ListTermsAdmin(ctx context.Context, req *ListTermsAdminReq) (res *ListTermsAdminRes, err error) {
	// 通过 machine_id 获取词汇表，再查询术语
	vocab, err := c.logic.GetVocabularyByMachineID(req.Vocabulary)
	if err != nil {
		return nil, err
	}
	terms, err := c.logic.GetTerms(vocab.ID, req.Hierarchy)
	if err != nil {
		return nil, err
	}
	return &ListTermsAdminRes{List: terms}, nil
}

// CreateTerm 创建术语
func (c *AdminController) CreateTerm(ctx context.Context, req *CreateTermReq) (res *CreateTermRes, err error) {
	// 通过 machine_id 获取词汇表，再创建术语
	vocab, err := c.logic.GetVocabularyByMachineID(req.Vocabulary)
	if err != nil {
		return nil, err
	}
	term, err := c.logic.CreateTerm(vocab.ID, req.ParentID, req.Name, req.Slug, req.Description, req.Weight)
	if err != nil {
		return nil, err
	}
	return &CreateTermRes{ID: term.ID}, nil
}

// UpdateTerm 更新术语
func (c *AdminController) UpdateTerm(ctx context.Context, req *UpdateTermReq) (res *UpdateTermRes, err error) {
	if err := c.logic.UpdateTerm(req.ID, req.Name, req.Slug, req.Description, req.Weight, req.ParentID); err != nil {
		return nil, err
	}
	return &UpdateTermRes{}, nil
}

// DeleteTerm 删除术语
func (c *AdminController) DeleteTerm(ctx context.Context, req *DeleteTermReq) (res *DeleteTermRes, err error) {
	if err := c.logic.DeleteTerm(req.ID); err != nil {
		return nil, err
	}
	return &DeleteTermRes{}, nil
}
