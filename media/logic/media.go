// Package logic media 业务逻辑
// 文件上传/存储/文件夹管理
package logic

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"gocms/core"
	"gocms/module/media/model"
)

// Logic media 业务逻辑
type Logic struct {
	db         *gorm.DB
	events     core.EventBus
	uploadPath string // 上传文件存储根路径
}

// NewLogic 创建 media 逻辑实例
func NewLogic(db *gorm.DB, events core.EventBus, uploadPath string) *Logic {
	return &Logic{db: db, events: events, uploadPath: uploadPath}
}

// allowedExtensions 允许上传的文件扩展名白名单
var allowedExtensions = map[string]bool{
	// 图片
	".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".bmp": true,
	".webp": true, ".svg": true, ".ico": true, ".tiff": true, ".tif": true,
	// 视频
	".mp4": true, ".avi": true, ".mov": true, ".wmv": true, ".flv": true,
	".mkv": true, ".webm": true, ".m4v": true,
	// 音频
	".mp3": true, ".wav": true, ".ogg": true, ".flac": true, ".aac": true,
	".wma": true, ".m4a": true,
	// 文档
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".txt": true, ".csv": true, ".rtf": true,
	".odt": true, ".ods": true, ".odp": true,
	// 压缩包
	".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true,
}

// isAllowedFileType 检查文件扩展名是否在白名单中
func isAllowedFileType(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return allowedExtensions[ext]
}

// ---------------------------------------------------------------------------
// 文件上传
// ---------------------------------------------------------------------------

// Upload 上传文件
func (l *Logic) Upload(file multipart.File, header *multipart.FileHeader, folderID *int64, userID int64) (*model.Media, error) {
	// 文件类型白名单校验：拒绝 .exe/.bat/.sh 等可执行文件
	if !isAllowedFileType(header.Filename) {
		return nil, fmt.Errorf("不允许上传该类型的文件：%s", filepath.Ext(header.Filename))
	}

	// 生成存储路径：/uploads/2026/03/filename.ext
	now := time.Now()
	dir := fmt.Sprintf("uploads/%d/%02d", now.Year(), now.Month())
	if err := os.MkdirAll(filepath.Join(l.uploadPath, dir), 0755); err != nil {
		return nil, fmt.Errorf("创建目录失败: %w", err)
	}

	// 安全文件名（时间戳 + 原始扩展名）
	ext := strings.ToLower(filepath.Ext(header.Filename))
	safeName := fmt.Sprintf("%d%s", now.UnixNano(), ext)
	storagePath := filepath.Join(dir, safeName)
	fullPath := filepath.Join(l.uploadPath, storagePath)

	// 写入磁盘
	dst, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("创建文件失败: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("写入文件失败: %w", err)
	}

	// 获取 MIME 类型
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// 创建数据库记录
	media := model.Media{
		FolderID:    folderID,
		Filename:    header.Filename,
		StoragePath: "/" + storagePath, // web 路径
		MimeType:    mimeType,
		Size:        header.Size,
		UploadedBy:  userID,
	}
	if err := l.db.Create(&media).Error; err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("保存记录失败: %w", err)
	}

	media.FillURL()
	l.events.EmitAsync("media.uploaded", core.MediaEvent{MediaID: media.ID, MimeType: mimeType})
	return &media, nil
}

// ---------------------------------------------------------------------------
// 媒体 CRUD
// ---------------------------------------------------------------------------

// List 媒体列表（分页 + 筛选）
// ownerID > 0 时仅返回该用户上传的媒体文件（scope:own）
func (l *Logic) List(folderID *int64, mimePrefix string, page, pageSize int, ownerID int64) ([]model.Media, int64, error) {
	var items []model.Media
	var total int64

	query := l.db.Model(&model.Media{})
	if folderID != nil {
		query = query.Where("folder_id = ?", *folderID)
	}
	if mimePrefix != "" {
		query = query.Where("mime_type LIKE ?", mimePrefix+"%")
	}
	if ownerID > 0 {
		query = query.Where("uploaded_by = ?", ownerID)
	}

	query.Count(&total)
	offset := (page - 1) * pageSize
	err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&items).Error
	if items == nil {
		items = make([]model.Media, 0)
	}
	for i := range items {
		items[i].FillURL()
	}
	return items, total, err
}

// GetByID 获取媒体详情
func (l *Logic) GetByID(id int64) (*model.Media, error) {
	var media model.Media
	if err := l.db.First(&media, id).Error; err != nil {
		return nil, fmt.Errorf("媒体文件不存在")
	}
	media.FillURL()
	return &media, nil
}

// Update 更新媒体元信息（Alt/Title）
func (l *Logic) Update(id int64, alt, title string) error {
	result := l.db.Model(&model.Media{}).Where("id = ?", id).Updates(map[string]interface{}{
		"alt":   alt,
		"title": title,
	})
	if result.RowsAffected == 0 {
		return fmt.Errorf("媒体文件不存在")
	}
	return result.Error
}

// Delete 删除媒体文件（软删除记录 + 删除磁盘文件）
func (l *Logic) Delete(id int64) error {
	var media model.Media
	if err := l.db.First(&media, id).Error; err != nil {
		return fmt.Errorf("媒体文件不存在")
	}

	// 软删除记录
	if err := l.db.Delete(&media).Error; err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	// 删除磁盘文件（异步，不阻塞响应）
	go func() {
		fullPath := filepath.Join(l.uploadPath, strings.TrimPrefix(media.StoragePath, "/"))
		os.Remove(fullPath)
	}()

	l.events.EmitAsync("media.deleted", core.MediaEvent{MediaID: id})
	return nil
}

// ---------------------------------------------------------------------------
// 文件夹管理
// ---------------------------------------------------------------------------

// ListFolders 获取文件夹树
func (l *Logic) ListFolders() ([]model.MediaFolder, error) {
	var folders []model.MediaFolder
	err := l.db.Where("parent_id IS NULL").Order("sort, id").
		Preload("Children", func(db *gorm.DB) *gorm.DB {
			return db.Order("sort, id")
		}).Find(&folders).Error
	if folders == nil {
		folders = make([]model.MediaFolder, 0)
	}
	return folders, err
}

// CreateFolder 创建文件夹
func (l *Logic) CreateFolder(name string, parentID *int64) (*model.MediaFolder, error) {
	folder := model.MediaFolder{Name: name, ParentID: parentID}
	if err := l.db.Create(&folder).Error; err != nil {
		return nil, fmt.Errorf("创建文件夹失败: %w", err)
	}
	return &folder, nil
}

// RenameFolder 重命名文件夹
func (l *Logic) RenameFolder(id int64, name string) error {
	result := l.db.Model(&model.MediaFolder{}).Where("id = ?", id).Update("name", name)
	if result.RowsAffected == 0 {
		return fmt.Errorf("文件夹不存在")
	}
	return result.Error
}

// DeleteFolder 删除文件夹（仅空文件夹可删）
func (l *Logic) DeleteFolder(id int64) error {
	// 检查是否有子文件夹或文件
	var childCount int64
	l.db.Model(&model.MediaFolder{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return fmt.Errorf("文件夹下有子文件夹，无法删除")
	}
	var fileCount int64
	l.db.Model(&model.Media{}).Where("folder_id = ?", id).Count(&fileCount)
	if fileCount > 0 {
		return fmt.Errorf("文件夹下有文件，无法删除")
	}

	return l.db.Delete(&model.MediaFolder{}, id).Error
}
