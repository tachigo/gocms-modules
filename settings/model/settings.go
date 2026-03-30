// Package model settings 数据模型
package model

// SiteConfig 站点配置（从 config/site.yaml 加载）
type SiteConfig struct {
	Name        string `yaml:"name" json:"name"`                // 站点名称
	Description string `yaml:"description" json:"description"`  // 站点描述
	URL         string `yaml:"url" json:"url"`                  // 站点 URL
	Logo        string `yaml:"logo" json:"logo"`                // Logo 路径
	Favicon     string `yaml:"favicon" json:"favicon"`          // Favicon 路径
	Language    string `yaml:"language" json:"language"`         // 默认语言
	Timezone    string `yaml:"timezone" json:"timezone"`         // 时区

	// SEO 默认值
	SEO SEOConfig `yaml:"seo" json:"seo"`

	// 分页配置
	Pagination PaginationConfig `yaml:"pagination" json:"pagination"`

	// 图片样式（供 media module 使用）
	ImageStyles map[string]ImageStyleConfig `yaml:"image_styles" json:"image_styles"`

	// 联系方式（管理员可见）
	Contact ContactConfig `yaml:"contact" json:"contact,omitempty"`
}

// SEOConfig SEO 默认配置
type SEOConfig struct {
	TitleSuffix string `yaml:"title_suffix" json:"title_suffix"` // 标题后缀
	Description string `yaml:"description" json:"description"`   // 默认描述
	Keywords    string `yaml:"keywords" json:"keywords"`         // 默认关键词
}

// PaginationConfig 分页配置
type PaginationConfig struct {
	DefaultPageSize int `yaml:"default_page_size" json:"default_page_size"` // 默认每页条数
	MaxPageSize     int `yaml:"max_page_size" json:"max_page_size"`         // 最大每页条数
}

// ImageStyleConfig 图片样式配置
type ImageStyleConfig struct {
	Width   int    `yaml:"width" json:"width"`
	Height  int    `yaml:"height" json:"height"`
	Mode    string `yaml:"mode" json:"mode"` // fit / fill / crop
	Quality int    `yaml:"quality" json:"quality"`
}

// ContactConfig 联系方式（敏感信息，仅管理员可见）
type ContactConfig struct {
	Email string `yaml:"email" json:"email"`
	Phone string `yaml:"phone" json:"phone"`
}

// PublicConfig 返回给公开 API 的配置（过滤敏感字段）
type PublicConfig struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description"`
	URL         string                     `json:"url"`
	Logo        string                     `json:"logo"`
	Favicon     string                     `json:"favicon"`
	Language    string                     `json:"language"`
	Timezone    string                     `json:"timezone"`
	SEO         SEOConfig                  `json:"seo"`
	Pagination  PaginationConfig           `json:"pagination"`
	ImageStyles map[string]ImageStyleConfig `json:"image_styles"`
}

// ToPublic 过滤敏感信息，生成公开配置
func (c *SiteConfig) ToPublic() *PublicConfig {
	return &PublicConfig{
		Name:        c.Name,
		Description: c.Description,
		URL:         c.URL,
		Logo:        c.Logo,
		Favicon:     c.Favicon,
		Language:    c.Language,
		Timezone:    c.Timezone,
		SEO:         c.SEO,
		Pagination:  c.Pagination,
		ImageStyles: c.ImageStyles,
	}
}
