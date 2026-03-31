// Package article 模块
package article

import (
	"gocms/internal/core"
)

func init() {
	core.Register(&Module{})
}

type Module struct{}

func (m *Module) Name() string        { return "article" }
func (m *Module) Description() string { return "模块" }
func (m *Module) Version() string     { return "1.0.0" }
func (m *Module) Dependencies() []string { return []string{} }
func (m *Module) Init(app *core.App) error { return nil }
func (m *Module) RegisterRoutes(rg *core.RouterGroup) {}
func (m *Module) Schema() core.ModuleSchema { return core.ModuleSchema{} }
