// Package logic permission 业务逻辑
// RBAC 角色权限管理：角色 CRUD、权限检查、用户角色管理
// 支持本地角色和 SSO 角色映射的并集策略
package logic

import (
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"

	"gocms/core"
	"gocms/module/permission/model"
)

// PermissionLogic 权限业务逻辑
type PermissionLogic struct {
	db       *gorm.DB
	events   core.EventBus
	schemas  map[string]core.ModuleSchema
	// roleMapping SSO 角色到本地角色的映射表
	// key: SSO 角色名, value: 本地角色名
	roleMapping map[string]string

	// 缓存（内存级，后续可接入 Redis）
	mu            sync.RWMutex
	roleCache     map[int64]*model.Role         // role_id → Role
	permCache     map[int64][]model.Permission  // role_id → Permissions
	userRoleCache map[int64][]int64             // user_id → role_ids
}

// Logic 是 PermissionLogic 的别名，用于 controller 引用
type Logic = PermissionLogic

// New 创建权限逻辑实例（别名函数）
func New(db *gorm.DB, events core.EventBus, schemas map[string]core.ModuleSchema, roleMapping map[string]string) *Logic {
	return NewPermissionLogic(db, events, schemas, roleMapping)
}

// NewPermissionLogic 创建权限逻辑实例
// roleMapping: SSO 角色到本地角色的映射表，可为 nil
func NewPermissionLogic(db *gorm.DB, events core.EventBus, schemas map[string]core.ModuleSchema, roleMapping map[string]string) *PermissionLogic {
	if roleMapping == nil {
		roleMapping = make(map[string]string)
	}
	return &PermissionLogic{
		db:            db,
		events:        events,
		schemas:       schemas,
		roleMapping:   roleMapping,
		roleCache:     make(map[int64]*model.Role),
		permCache:     make(map[int64][]model.Permission),
		userRoleCache: make(map[int64][]int64),
	}
}

// ---------------------------------------------------------------------------
// 默认角色初始化
// ---------------------------------------------------------------------------

// SeedDefaultRoles 初始化默认角色（别名）
func (l *PermissionLogic) SeedDefaultRoles() error {
	return l.InitDefaultRoles()
}

// InitDefaultRoles 初始化 3 个默认角色（admin/editor/viewer）
func (l *PermissionLogic) InitDefaultRoles() error {
	// 检查是否已有角色
	var count int64
	if err := l.db.Model(&model.Role{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有角色，跳过
	}

	// 创建默认角色
	defaultRoles := []model.Role{
		{Name: "admin", Label: "管理员", Description: "系统管理员，拥有所有权限", IsSystem: true},
		{Name: "editor", Label: "编辑", Description: "内容编辑，可管理文章、页面、分类、媒体、菜单", IsSystem: true},
		{Name: "author", Label: "作者", Description: "内容作者，可读写自己的内容", IsSystem: true},
		{Name: "viewer", Label: "访客", Description: "普通访客，仅可查看内容", IsSystem: true},
	}

	for _, role := range defaultRoles {
		if err := l.db.Create(&role).Error; err != nil {
			return fmt.Errorf("创建默认角色 %s 失败: %w", role.Name, err)
		}
	}

	// 为 admin 角色分配所有权限
	if err := l.initAdminPermissions(); err != nil {
		return err
	}

	// 为 editor 角色分配内容管理权限（read:all + write:all）
	if err := l.initEditorPermissions(); err != nil {
		return err
	}

	// 为 author 角色分配自有内容权限（read:own + write:own）
	if err := l.initAuthorPermissions(); err != nil {
		return err
	}

	// 为 viewer 角色分配查看权限（read:all）
	if err := l.initViewerPermissions(); err != nil {
		return err
	}

	// 为种子用户分配角色（admin→admin, editor→editor, author→author）
	if err := l.assignSeedUserRoles(); err != nil {
		return err
	}

	return nil
}

// initAdminPermissions 为 admin 角色分配所有模块的所有权限
func (l *PermissionLogic) initAdminPermissions() error {
	var adminRole model.Role
	if err := l.db.Where("name = ?", "admin").First(&adminRole).Error; err != nil {
		return err
	}

	// 遍历所有模块，分配所有权限
	for moduleName, schema := range l.schemas {
		for _, perm := range schema.Permissions {
			// admin 拥有所有数据范围
			scopes := []string{"all"}
			if len(perm.Scopes) > 0 {
				scopes = perm.Scopes
			}
			for _, scope := range scopes {
				p := model.Permission{
					RoleID: adminRole.ID,
					Module: moduleName,
					Action: perm.Action,
					Scope:  scope,
				}
				if err := l.db.Create(&p).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// initEditorPermissions 为 editor 角色分配内容管理权限（read:all + write:all）
func (l *PermissionLogic) initEditorPermissions() error {
	var editorRole model.Role
	if err := l.db.Where("name = ?", "editor").First(&editorRole).Error; err != nil {
		return err
	}

	// editor 可管理的内容模块：article/page/taxonomy/media/menu
	contentModules := []string{"article", "page", "taxonomy", "media", "menu"}

	for _, module := range contentModules {
		schema, exists := l.schemas[module]
		if !exists {
			continue
		}

		for _, perm := range schema.Permissions {
			// editor 对内容模块拥有 read:all + write:all 权限
			p := model.Permission{
				RoleID: editorRole.ID,
				Module: module,
				Action: perm.Action,
				Scope:  "all",
			}
			if err := l.db.Create(&p).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// initAuthorPermissions 为 author 角色分配自有内容权限（read:own + write:own）
func (l *PermissionLogic) initAuthorPermissions() error {
	var authorRole model.Role
	if err := l.db.Where("name = ?", "author").First(&authorRole).Error; err != nil {
		return err
	}

	// author 可操作的内容模块
	contentModules := []string{"article", "page", "media", "taxonomy", "menu"}

	for _, module := range contentModules {
		schema, exists := l.schemas[module]
		if !exists {
			continue
		}

		for _, perm := range schema.Permissions {
			// author 对内容模块拥有 read:own + write:own 权限
			p := model.Permission{
				RoleID: authorRole.ID,
				Module: module,
				Action: perm.Action,
				Scope:  "own",
			}
			if err := l.db.Create(&p).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// initViewerPermissions 为 viewer 角色分配查看权限
func (l *PermissionLogic) initViewerPermissions() error {
	var viewerRole model.Role
	if err := l.db.Where("name = ?", "viewer").First(&viewerRole).Error; err != nil {
		return err
	}

	// viewer 仅可查看公开内容
	publicModules := []string{"article", "page", "taxonomy", "menu"}

	for _, module := range publicModules {
		// 只授予 read 权限
		p := model.Permission{
			RoleID: viewerRole.ID,
			Module: module,
			Action: "read",
			Scope:  "all",
		}
		if err := l.db.Create(&p).Error; err != nil {
			return err
		}
	}

	return nil
}

// assignSeedUserRoles 为种子用户分配角色
// 映射关系：admin→admin角色, editor→editor角色, author→author角色
func (l *PermissionLogic) assignSeedUserRoles() error {
	// username → role name 映射
	seedMapping := []struct {
		Username string
		RoleName string
	}{
		{"admin", "admin"},
		{"editor", "editor"},
		{"author", "author"},
	}

	for _, sm := range seedMapping {
		// 查询用户 ID（直接查 users 表，避免循环依赖 user model）
		var userID int64
		row := l.db.Table("users").Select("id").Where("username = ?", sm.Username).Row()
		if err := row.Scan(&userID); err != nil {
			// 用户不存在则跳过（可能 user module 未创建该用户）
			continue
		}

		// 查询角色 ID
		var role model.Role
		if err := l.db.Where("name = ?", sm.RoleName).First(&role).Error; err != nil {
			continue
		}

		// 检查是否已存在关联
		var count int64
		l.db.Model(&model.UserRole{}).Where("user_id = ? AND role_id = ?", userID, role.ID).Count(&count)
		if count > 0 {
			continue
		}

		// 创建用户-角色关联
		ur := model.UserRole{
			UserID: userID,
			RoleID: role.ID,
		}
		if err := l.db.Create(&ur).Error; err != nil {
			return fmt.Errorf("分配角色 %s 给用户 %s 失败: %w", sm.RoleName, sm.Username, err)
		}
	}

	return nil
}

// ---------------------------------------------------------------------------
// 角色 CRUD
// ---------------------------------------------------------------------------

// ListRoles 角色列表
func (l *PermissionLogic) ListRoles() ([]model.Role, error) {
	var roles []model.Role
	err := l.db.Find(&roles).Error
	if roles == nil {
		roles = make([]model.Role, 0)
	}
	return roles, err
}

// GetRole 获取角色详情（含权限列表）
func (l *PermissionLogic) GetRole(id int64) (*model.RoleWithPermissions, error) {
	var role model.Role
	if err := l.db.First(&role, id).Error; err != nil {
		return nil, fmt.Errorf("角色不存在")
	}

	var permissions []model.Permission
	if err := l.db.Where("role_id = ?", id).Find(&permissions).Error; err != nil {
		return nil, err
	}

	return &model.RoleWithPermissions{
		Role:        role,
		Permissions: permissions,
	}, nil
}

// CreateRole 创建角色
func (l *PermissionLogic) CreateRole(name, label, description string, permissions []model.Permission) (*model.Role, error) {
	// 检查角色名是否已存在
	var count int64
	l.db.Model(&model.Role{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("角色名已存在")
	}

	role := model.Role{
		Name:        name,
		Label:       label,
		Description: description,
		IsSystem:    false, // 自定义角色
	}

	if err := l.db.Create(&role).Error; err != nil {
		return nil, fmt.Errorf("创建角色失败: %w", err)
	}

	// 创建权限
	for _, perm := range permissions {
		perm.RoleID = role.ID
		if err := l.db.Create(&perm).Error; err != nil {
			return nil, err
		}
	}

	// 清除缓存
	l.clearRoleCache(role.ID)

	// 发布事件
	l.events.EmitAsync("permission.role_created", core.RoleEvent{RoleID: role.ID})

	return &role, nil
}

// UpdateRole 更新角色权限
func (l *PermissionLogic) UpdateRole(id int64, label, description string, permissions []model.Permission) error {
	var role model.Role
	if err := l.db.First(&role, id).Error; err != nil {
		return fmt.Errorf("角色不存在")
	}

	// 系统角色不能修改基本信息（但可以修改权限）
	if role.IsSystem && (label != "" || description != "") {
		// 系统角色只允许修改权限，不允许修改名称/描述
		// 这里我们允许修改 label 和 description，但不允许修改 name
	}

	// 更新角色信息
	updates := map[string]interface{}{}
	if label != "" {
		updates["label"] = label
	}
	if description != "" {
		updates["description"] = description
	}
	if len(updates) > 0 {
		if err := l.db.Model(&role).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新角色失败: %w", err)
		}
	}

	// 如果提供了权限列表，则更新权限
	if permissions != nil {
		// 删除旧权限
		if err := l.db.Where("role_id = ?", id).Delete(&model.Permission{}).Error; err != nil {
			return err
		}

		// 创建新权限
		for _, perm := range permissions {
			perm.RoleID = id
			if err := l.db.Create(&perm).Error; err != nil {
				return err
			}
		}
	}

	// 清除缓存
	l.clearRoleCache(id)

	// 发布事件
	l.events.EmitAsync("permission.role_updated", core.RoleEvent{RoleID: id})

	return nil
}

// DeleteRole 删除角色
func (l *PermissionLogic) DeleteRole(id int64) error {
	var role model.Role
	if err := l.db.First(&role, id).Error; err != nil {
		return fmt.Errorf("角色不存在")
	}

	// 系统角色不能删除
	if role.IsSystem {
		return fmt.Errorf("系统角色不能删除")
	}

	// 检查是否有用户关联此角色
	var userCount int64
	l.db.Model(&model.UserRole{}).Where("role_id = ?", id).Count(&userCount)
	if userCount > 0 {
		return fmt.Errorf("该角色仍有用户关联，不能删除")
	}

	// 删除角色权限
	if err := l.db.Where("role_id = ?", id).Delete(&model.Permission{}).Error; err != nil {
		return err
	}

	// 删除角色
	if err := l.db.Delete(&role).Error; err != nil {
		return err
	}

	// 清除缓存
	l.clearRoleCache(id)

	// 发布事件
	l.events.EmitAsync("permission.role_deleted", core.RoleEvent{RoleID: id})

	return nil
}

// ---------------------------------------------------------------------------
// 用户角色管理
// ---------------------------------------------------------------------------

// GetUserRoles 获取用户的角色列表
func (l *PermissionLogic) GetUserRoles(userID int64) ([]model.Role, error) {
	// 检查缓存
	l.mu.RLock()
	roleIDs, exists := l.userRoleCache[userID]
	l.mu.RUnlock()

	if exists {
		var roles []model.Role
		if len(roleIDs) > 0 {
			l.db.Where("id IN ?", roleIDs).Find(&roles)
		}
		if roles == nil {
			roles = make([]model.Role, 0)
		}
		return roles, nil
	}

	// 从数据库查询
	var userRoles []model.UserRole
	if err := l.db.Where("user_id = ?", userID).Find(&userRoles).Error; err != nil {
		return nil, err
	}

	roleIDs = make([]int64, 0, len(userRoles))
	for _, ur := range userRoles {
		roleIDs = append(roleIDs, ur.RoleID)
	}

	var roles []model.Role
	if len(roleIDs) > 0 {
		l.db.Where("id IN ?", roleIDs).Find(&roles)
	}
	if roles == nil {
		roles = make([]model.Role, 0)
	}

	// 写入缓存
	l.mu.Lock()
	l.userRoleCache[userID] = roleIDs
	l.mu.Unlock()

	return roles, nil
}

// AssignRolesToUser 为用户分配角色
func (l *PermissionLogic) AssignRolesToUser(userID int64, roleIDs []int64) error {
	// 删除旧的角色关联
	if err := l.db.Where("user_id = ?", userID).Delete(&model.UserRole{}).Error; err != nil {
		return err
	}

	// 创建新的角色关联
	for _, roleID := range roleIDs {
		ur := model.UserRole{
			UserID: userID,
			RoleID: roleID,
		}
		if err := l.db.Create(&ur).Error; err != nil {
			return err
		}
	}

	// 清除缓存
	l.mu.Lock()
	delete(l.userRoleCache, userID)
	l.mu.Unlock()

	return nil
}

// AddRoleToUser 为用户添加单个角色
func (l *PermissionLogic) AddRoleToUser(userID, roleID int64) error {
	// 检查是否已存在
	var count int64
	l.db.Model(&model.UserRole{}).Where("user_id = ? AND role_id = ?", userID, roleID).Count(&count)
	if count > 0 {
		return nil // 已存在，无需重复添加
	}

	ur := model.UserRole{
		UserID: userID,
		RoleID: roleID,
	}
	if err := l.db.Create(&ur).Error; err != nil {
		return err
	}

	// 清除缓存
	l.mu.Lock()
	delete(l.userRoleCache, userID)
	l.mu.Unlock()

	return nil
}

// RemoveRoleFromUser 移除用户的某个角色
func (l *PermissionLogic) RemoveRoleFromUser(userID, roleID int64) error {
	if err := l.db.Where("user_id = ? AND role_id = ?", userID, roleID).Delete(&model.UserRole{}).Error; err != nil {
		return err
	}

	// 清除缓存
	l.mu.Lock()
	delete(l.userRoleCache, userID)
	l.mu.Unlock()

	return nil
}

// ---------------------------------------------------------------------------
// 权限检查（核心）
// ---------------------------------------------------------------------------

// CheckPermission 检查用户是否有指定权限
// 采用并集策略：本地角色权限 + SSO 角色映射权限
// userID: 用户ID
// module: 模块名
// action: 操作
// ssoRole: SSO 角色名（可为空，支持逗号分隔的多角色，如 "sso_manager,sso_editor"）
// returns: hasPermission, scope (own/all), error
func (l *PermissionLogic) CheckPermission(userID int64, module, action, ssoRole string) (bool, string, error) {
	// 优先检查 SSO 角色映射（性能优化：减少数据库 IO）
	// 支持逗号分隔的多角色格式
	if ssoRole != "" {
		ssoRoles := strings.Split(ssoRole, ",")
		for _, role := range ssoRoles {
			role = strings.TrimSpace(role)
			if role == "" {
				continue
			}
			if localRoleName, exists := l.roleMapping[role]; exists {
				// 查询映射的本地角色
				var mappedRole model.Role
				if err := l.db.Where("name = ?", localRoleName).First(&mappedRole).Error; err == nil {
					// 检查映射角色的权限
					if mappedRole.Name == "admin" {
						return true, "all", nil
					}
					hasPerm, scope := l.checkRolePermission(mappedRole.ID, module, action)
					if hasPerm {
						return true, scope, nil
					}
				}
			}
		}
	}

	// 获取用户的本地角色
	roles, err := l.GetUserRoles(userID)
	if err != nil {
		return false, "", err
	}

	// 检查本地角色的权限
	for _, role := range roles {
		// admin 角色拥有所有权限
		if role.Name == "admin" {
			return true, "all", nil
		}

		// 检查角色权限
		hasPerm, scope := l.checkRolePermission(role.ID, module, action)
		if hasPerm {
			return true, scope, nil
		}
	}

	return false, "", nil
}

// CheckPermissionWithContext 检查用户权限（带上下文）
// 从 Context 中自动提取 UserInfo.Role 进行 SSO 角色映射
func (l *PermissionLogic) CheckPermissionWithContext(ctx core.Context, userID int64, module, action string) (bool, string, error) {
	// 从 Context 获取 SSO 角色
	userInfo := core.GetUserFromCtx(ctx)
	ssoRole := ""
	if userInfo != nil {
		ssoRole = userInfo.Role
	}

	return l.CheckPermission(userID, module, action, ssoRole)
}

// checkRolePermission 检查角色是否有指定权限（内部方法，带缓存）
func (l *PermissionLogic) checkRolePermission(roleID int64, module, action string) (bool, string) {
	// 检查缓存
	l.mu.RLock()
	perms, exists := l.permCache[roleID]
	l.mu.RUnlock()

	if !exists {
		// 从数据库加载
		l.db.Where("role_id = ?", roleID).Find(&perms)
		l.mu.Lock()
		l.permCache[roleID] = perms
		l.mu.Unlock()
	}

	// 遍历权限列表
	for _, perm := range perms {
		if perm.Module == module && perm.Action == action {
			return true, perm.Scope
		}
		// manage 权限包含所有操作
		if perm.Module == module && perm.Action == "manage" {
			return true, perm.Scope
		}
	}

	return false, ""
}

// IsAdmin 检查用户是否是管理员
// 采用并集策略：本地角色为 admin 或 SSO 角色映射到 admin
// ssoRole: 支持逗号分隔的多角色，如 "sso_manager,sso_editor"
func (l *PermissionLogic) IsAdmin(userID int64, ssoRole string) bool {
	// 优先检查 SSO 角色映射（性能优化）
	if ssoRole != "" {
		ssoRoles := strings.Split(ssoRole, ",")
		for _, role := range ssoRoles {
			role = strings.TrimSpace(role)
			if role == "" {
				continue
			}
			if localRoleName, exists := l.roleMapping[role]; exists {
				if localRoleName == "admin" {
					return true
				}
			}
		}
	}

	// 检查本地角色
	roles, err := l.GetUserRoles(userID)
	if err != nil {
		return false
	}

	for _, role := range roles {
		if role.Name == "admin" {
			return true
		}
	}

	return false
}

// IsAdminWithContext 检查用户是否是管理员（带上下文）
// 从 Context 中自动提取 UserInfo.Role 进行 SSO 角色映射
func (l *PermissionLogic) IsAdminWithContext(ctx core.Context, userID int64) bool {
	// 从 Context 获取 SSO 角色
	userInfo := core.GetUserFromCtx(ctx)
	ssoRole := ""
	if userInfo != nil {
		ssoRole = userInfo.Role
	}

	return l.IsAdmin(userID, ssoRole)
}

// GetAllAvailablePermissions 获取所有可用权限（来自各 Module Schema）
func (l *PermissionLogic) GetAllAvailablePermissions() []PermissionGroup {
	var groups []PermissionGroup

	for moduleName, schema := range l.schemas {
		group := PermissionGroup{
			Module:      moduleName,
			Permissions: schema.Permissions,
		}
		groups = append(groups, group)
	}

	return groups
}

// PermissionGroup 权限分组（按模块）
type PermissionGroup struct {
	Module      string               `json:"module"`
	Permissions []core.PermissionDef `json:"permissions"`
}

// ---------------------------------------------------------------------------
// 缓存管理
// ---------------------------------------------------------------------------

// clearRoleCache 清除角色相关缓存
func (l *PermissionLogic) clearRoleCache(roleID int64) {
	l.mu.Lock()
	delete(l.roleCache, roleID)
	delete(l.permCache, roleID)
	l.mu.Unlock()
}

// RefreshCache 刷新所有缓存（在权限变更时调用）
func (l *PermissionLogic) RefreshCache() {
	l.mu.Lock()
	l.roleCache = make(map[int64]*model.Role)
	l.permCache = make(map[int64][]model.Permission)
	l.userRoleCache = make(map[int64][]int64)
	l.mu.Unlock()
}
