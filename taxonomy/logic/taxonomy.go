// Package logic taxonomy 业务逻辑
// 词汇表/术语 CRUD 操作
package logic

import (
	"fmt"

	"gorm.io/gorm"

	"gocms/core"
	"gocms/module/taxonomy/model"
)

// Logic taxonomy 业务逻辑
type Logic struct {
	db     *gorm.DB
	events core.EventBus
}

// NewLogic 创建 taxonomy 逻辑实例
func NewLogic(db *gorm.DB, events core.EventBus) *Logic {
	return &Logic{db: db, events: events}
}

// ---------------------------------------------------------------------------
// 词汇表（Vocabulary）管理
// ---------------------------------------------------------------------------

// ListVocabularies 获取所有词汇表列表
func (l *Logic) ListVocabularies() ([]model.Vocabulary, error) {
	var vocabularies []model.Vocabulary
	err := l.db.Order("weight DESC, id ASC").Find(&vocabularies).Error
	if vocabularies == nil {
		vocabularies = make([]model.Vocabulary, 0)
	}
	return vocabularies, err
}

// GetVocabularyByMachineID 通过机器名获取词汇表
func (l *Logic) GetVocabularyByMachineID(machineID string) (*model.Vocabulary, error) {
	var vocab model.Vocabulary
	if err := l.db.Where("machine_id = ?", machineID).First(&vocab).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("词汇表不存在")
		}
		return nil, err
	}
	return &vocab, nil
}

// ---------------------------------------------------------------------------
// 术语（Term）管理
// ---------------------------------------------------------------------------

// GetTerms 获取指定词汇表下的所有术语
// 如果 hierarchy 为 true，返回树形结构；否则返回平级列表
func (l *Logic) GetTerms(vocabularyID int64, hierarchy bool) ([]model.Term, error) {
	var terms []model.Term
	query := l.db.Where("vocabulary_id = ?", vocabularyID)

	if hierarchy {
		// 树形结构：只获取根级术语（ParentID IS NULL），预加载子术语
		err := query.Where("parent_id IS NULL").
			Order("weight DESC, id ASC").
			Preload("Children", func(db *gorm.DB) *gorm.DB {
				return db.Order("weight DESC, id ASC")
			}).Find(&terms).Error
		if terms == nil {
			terms = make([]model.Term, 0)
		}
		return terms, err
	}

	// 平级列表：获取所有术语
	err := query.Order("weight DESC, id ASC").Find(&terms).Error
	if terms == nil {
		terms = make([]model.Term, 0)
	}
	return terms, err
}

// GetTermByID 通过 ID 获取术语
func (l *Logic) GetTermByID(id int64) (*model.Term, error) {
	var term model.Term
	if err := l.db.First(&term, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("术语不存在")
		}
		return nil, err
	}
	return &term, nil
}

// GetTermBySlug 通过 slug 和词汇表获取术语
func (l *Logic) GetTermBySlug(vocabularyID int64, slug string) (*model.Term, error) {
	var term model.Term
	if err := l.db.Where("vocabulary_id = ? AND slug = ?", vocabularyID, slug).First(&term).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("术语不存在")
		}
		return nil, err
	}
	return &term, nil
}

// CreateTerm 创建术语
func (l *Logic) CreateTerm(vocabularyID int64, parentID *int64, name, slug, description string, weight int) (*model.Term, error) {
	// 检查词汇表是否存在
	var vocab model.Vocabulary
	if err := l.db.First(&vocab, vocabularyID).Error; err != nil {
		return nil, fmt.Errorf("词汇表不存在")
	}

	// 检查 slug 是否唯一（在同一词汇表内）
	var count int64
	l.db.Model(&model.Term{}).Where("vocabulary_id = ? AND slug = ?", vocabularyID, slug).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("术语标识 '%s' 已存在", slug)
	}

	// 如果指定了父级，检查父级是否属于同一词汇表
	if parentID != nil {
		var parent model.Term
		if err := l.db.First(&parent, *parentID).Error; err != nil {
			return nil, fmt.Errorf("父级术语不存在")
		}
		if parent.VocabularyID != vocabularyID {
			return nil, fmt.Errorf("父级术语不属于该词汇表")
		}
		// 检查词汇表是否支持层级
		if !vocab.Hierarchy {
			return nil, fmt.Errorf("该词汇表不支持层级分类")
		}
	}

	term := model.Term{
		VocabularyID: vocabularyID,
		ParentID:     parentID,
		Name:         name,
		Slug:         slug,
		Description:  description,
		Weight:       weight,
	}

	if err := l.db.Create(&term).Error; err != nil {
		return nil, fmt.Errorf("创建术语失败: %w", err)
	}

	l.events.EmitAsync("taxonomy.term_created", core.ContentEvent{
		Module: "taxonomy",
		ID:     term.ID,
		Data:   term,
	})

	return &term, nil
}

// UpdateTerm 更新术语
func (l *Logic) UpdateTerm(id int64, name, slug, description string, weight int, parentID *int64) error {
	var term model.Term
	if err := l.db.First(&term, id).Error; err != nil {
		return fmt.Errorf("术语不存在")
	}

	// 如果修改 slug，检查唯一性
	if slug != "" && slug != term.Slug {
		var count int64
		l.db.Model(&model.Term{}).Where("vocabulary_id = ? AND slug = ? AND id != ?", term.VocabularyID, slug, id).Count(&count)
		if count > 0 {
			return fmt.Errorf("术语标识 '%s' 已存在", slug)
		}
	}

	// 如果修改 parentID，检查是否形成循环引用
	if parentID != nil && term.ParentID != parentID {
		if *parentID == id {
			return fmt.Errorf("不能将自己设为父级")
		}
		var parent model.Term
		if err := l.db.First(&parent, *parentID).Error; err != nil {
			return fmt.Errorf("父级术语不存在")
		}
		if parent.VocabularyID != term.VocabularyID {
			return fmt.Errorf("父级术语不属于同一词汇表")
		}
		// 检查词汇表是否支持层级
		var vocab model.Vocabulary
		l.db.First(&vocab, term.VocabularyID)
		if !vocab.Hierarchy {
			return fmt.Errorf("该词汇表不支持层级分类")
		}
	}

	updates := map[string]interface{}{}
	if name != "" {
		updates["name"] = name
	}
	if slug != "" {
		updates["slug"] = slug
	}
	updates["description"] = description
	updates["weight"] = weight
	if parentID != nil {
		updates["parent_id"] = *parentID
	}

	result := l.db.Model(&model.Term{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新术语失败: %w", result.Error)
	}

	l.events.EmitAsync("taxonomy.term_updated", core.ContentEvent{
		Module: "taxonomy",
		ID:     id,
		Data:   updates,
	})

	return nil
}

// DeleteTerm 删除术语（软删除）
// 如果术语有子术语，不能删除
func (l *Logic) DeleteTerm(id int64) error {
	var term model.Term
	if err := l.db.First(&term, id).Error; err != nil {
		return fmt.Errorf("术语不存在")
	}

	// 检查是否有子术语
	var childCount int64
	l.db.Model(&model.Term{}).Where("parent_id = ?", id).Count(&childCount)
	if childCount > 0 {
		return fmt.Errorf("该术语下有子术语，请先删除子术语")
	}

	if err := l.db.Delete(&term).Error; err != nil {
		return fmt.Errorf("删除术语失败: %w", err)
	}

	l.events.EmitAsync("taxonomy.term_deleted", core.ContentEvent{
		Module: "taxonomy",
		ID:     id,
	})

	return nil
}

// GetTermsByVocabularyMachineID 通过词汇表机器名获取术语列表
// 这是公开 API 使用的方法
func (l *Logic) GetTermsByVocabularyMachineID(machineID string, hierarchy bool) ([]model.Term, error) {
	vocab, err := l.GetVocabularyByMachineID(machineID)
	if err != nil {
		return nil, err
	}
	return l.GetTerms(vocab.ID, hierarchy)
}

// GetTermByVocabularyAndID 通过词汇表机器名和术语 ID 获取术语详情
// 这是公开 API 使用的方法
func (l *Logic) GetTermByVocabularyAndID(machineID string, termID int64) (*model.Term, error) {
	vocab, err := l.GetVocabularyByMachineID(machineID)
	if err != nil {
		return nil, err
	}

	var term model.Term
	if err := l.db.Where("id = ? AND vocabulary_id = ?", termID, vocab.ID).First(&term).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("术语不存在")
		}
		return nil, err
	}
	return &term, nil
}
