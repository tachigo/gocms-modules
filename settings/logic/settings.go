// Package logic settings 业务逻辑
// 从 config/site.yaml 加载站点配置，提供读取接口
package logic

import (
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"

	"gocms/module/settings/model"
)

// Logic settings 业务逻辑
type Logic struct {
	config *model.SiteConfig
	mu     sync.RWMutex
}

// NewLogic 创建 settings 逻辑实例
func NewLogic() *Logic {
	return &Logic{}
}

// LoadFromFile 从 YAML 文件加载站点配置
func (l *Logic) LoadFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config model.SiteConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 设置默认值
	if config.Language == "" {
		config.Language = "zh-CN"
	}
	if config.Timezone == "" {
		config.Timezone = "Asia/Shanghai"
	}
	if config.Pagination.DefaultPageSize == 0 {
		config.Pagination.DefaultPageSize = 20
	}
	if config.Pagination.MaxPageSize == 0 {
		config.Pagination.MaxPageSize = 100
	}

	l.mu.Lock()
	l.config = &config
	l.mu.Unlock()
	return nil
}

// GetPublicConfig 获取公开配置（过滤敏感信息）
func (l *Logic) GetPublicConfig() *model.PublicConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.config == nil {
		return &model.PublicConfig{}
	}
	return l.config.ToPublic()
}

// GetFullConfig 获取完整配置（含敏感信息，仅管理员）
func (l *Logic) GetFullConfig() *model.SiteConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.config == nil {
		return &model.SiteConfig{}
	}
	return l.config
}

// GetImageStyles 获取图片样式配置（供 media module 使用）
func (l *Logic) GetImageStyles() map[string]model.ImageStyleConfig {
	l.mu.RLock()
	defer l.mu.RUnlock()
	if l.config == nil {
		return nil
	}
	return l.config.ImageStyles
}
