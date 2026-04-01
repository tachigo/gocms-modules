// Package logic user 业务逻辑
// 用户认证、个人信息管理、用户 CRUD
// 支持两种运行模式：
//   - master: 独立运行，拥有完整的用户管理功能（登录/注册/数据库）
//   - slave:  作为SSO客户端运行，依赖上游SSO系统鉴权，不维护本地用户表
package logic

import (
	"context"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"gocms/core"
	"gocms/module/user/model"
)

// SlaveModeError 表示在 slave 模式下禁止的操作
// 用于向前端返回 403 状态码和友好的错误信息
type SlaveModeError struct {
	Message string
}

func (e *SlaveModeError) Error() string {
	return e.Message
}

// HTTPStatus 返回 HTTP 状态码 403 Forbidden
func (e *SlaveModeError) HTTPStatus() int {
	return http.StatusForbidden
}

// UserLogic 用户业务逻辑
type UserLogic struct {
	db     *gorm.DB
	jwt    *JWTManager
	events core.EventBus
	mode   string // 运行模式: master | slave
}

// NewUserLogic 创建用户逻辑实例
// mode 参数指定运行模式：master 为独立模式，slave 为SSO从属模式
func NewUserLogic(db *gorm.DB, jwt *JWTManager, events core.EventBus, mode string) *UserLogic {
	return &UserLogic{db: db, jwt: jwt, events: events, mode: mode}
}

// JWTManager 获取 JWT 管理器（供中间件使用）
func (l *UserLogic) JWTManager() *JWTManager {
	return l.jwt
}

// ---------------------------------------------------------------------------
// 认证相关
// ---------------------------------------------------------------------------

// Login 用户登录，验证成功返回 JWT Token
// 仅在 master 模式下可用，slave 模式下登录由SSO系统处理
func (l *UserLogic) Login(username, password string) (string, *model.User, error) {
	// slave 模式下禁止本地登录
	if l.mode == "slave" {
		return "", nil, &SlaveModeError{Message: "当前系统处于SSO从属模式，请使用SSO系统登录"}
	}

	var user model.User
	if err := l.db.Where("username = ?", username).First(&user).Error; err != nil {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}

	// 检查用户状态
	if user.Status != "active" {
		return "", nil, fmt.Errorf("账号已被禁用")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", nil, fmt.Errorf("用户名或密码错误")
	}

	// 生成 JWT Token
	token, err := l.jwt.GenerateToken(user.ID, user.Username)
	if err != nil {
		return "", nil, fmt.Errorf("生成Token失败: %w", err)
	}

	// 发布登录事件
	l.events.EmitAsync("user.login", core.UserEvent{UserID: user.ID})

	return token, &user, nil
}

// Logout 用户登出，将 Token 加入黑名单
func (l *UserLogic) Logout(token string) {
	l.jwt.AddToBlacklist(token)
}

// ---------------------------------------------------------------------------
// 个人信息
// ---------------------------------------------------------------------------

// GetProfile 获取用户个人信息
// 根据运行模式决定数据来源：
//   - master: 从本地数据库查询
//   - slave:  从 Context 中获取（由SSO中间件注入）
func (l *UserLogic) GetProfile(ctx context.Context, userID int64) (*model.User, error) {
	// slave 模式：优先从 Context 获取，不回表查询
	if l.mode == "slave" {
		userInfo := core.GetUserFromCtx(ctx)
		if userInfo == nil {
			return nil, fmt.Errorf("上下文中无用户信息")
		}
		return &model.User{
			ID:       userInfo.ID,
			Username: userInfo.Username,
			Email:    userInfo.Email,
			// slave 模式下其他字段可能为空，由SSO系统决定
		}, nil
	}

	// master 模式：从本地数据库查询
	var user model.User
	if err := l.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return &user, nil
}

// UpdateProfile 更新个人信息（昵称、头像）
// 仅在 master 模式下支持修改，slave 模式下应由SSO系统管理用户信息
func (l *UserLogic) UpdateProfile(userID int64, nickname, avatar string) error {
	// slave 模式下禁止修改用户信息，应由SSO系统统一管理
	if l.mode == "slave" {
		return &SlaveModeError{Message: "SSO从属模式下用户信息由SSO系统统一管理，禁止本地修改"}
	}

	updates := map[string]interface{}{}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if avatar != "" {
		updates["avatar"] = avatar
	}
	if len(updates) == 0 {
		return nil
	}

	result := l.db.Model(&model.User{}).Where("id = ?", userID).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新失败: %w", result.Error)
	}

	l.events.EmitAsync("user.updated", core.UserEvent{UserID: userID})
	return nil
}

// ChangePassword 修改密码
// 仅在 master 模式下可用，slave 模式下密码管理由SSO系统处理
func (l *UserLogic) ChangePassword(userID int64, oldPassword, newPassword string) error {
	// slave 模式下禁止修改密码，应由SSO系统统一管理
	if l.mode == "slave" {
		return &SlaveModeError{Message: "SSO从属模式下密码由SSO系统统一管理，禁止本地修改"}
	}

	var user model.User
	if err := l.db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("用户不存在")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(oldPassword)); err != nil {
		return fmt.Errorf("旧密码错误")
	}

	// 哈希新密码
	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("密码加密失败: %w", err)
	}

	if err := l.db.Model(&user).Update("password", string(hashed)).Error; err != nil {
		return fmt.Errorf("修改密码失败: %w", err)
	}

	// 使该用户所有旧 Token 失效
	l.jwt.InvalidateUser(userID)

	l.events.EmitAsync("user.password_changed", core.UserEvent{UserID: userID})
	return nil
}

// ---------------------------------------------------------------------------
// 用户管理（管理员）
// ---------------------------------------------------------------------------

// List 用户列表（分页）
// slave 模式下返回空列表（无本地用户表）
func (l *UserLogic) List(page, pageSize int) ([]model.User, int64, error) {
	// slave 模式：无本地用户表，返回空列表
	if l.mode == "slave" {
		return []model.User{}, 0, nil
	}

	var users []model.User
	var total int64

	l.db.Model(&model.User{}).Count(&total)

	offset := (page - 1) * pageSize
	err := l.db.Offset(offset).Limit(pageSize).Order("id DESC").Find(&users).Error
	if users == nil {
		users = make([]model.User, 0)
	}
	return users, total, err
}

// Create 创建用户
// 仅在 master 模式下可用
func (l *UserLogic) Create(username, email, password, nickname string) (*model.User, error) {
	// slave 模式下禁止创建本地用户
	if l.mode == "slave" {
		return nil, &SlaveModeError{Message: "SSO从属模式下禁止创建本地用户，请使用SSO系统管理用户"}
	}

	// 检查用户名唯一
	var count int64
	l.db.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("用户名已存在")
	}

	// 检查邮箱唯一
	l.db.Model(&model.User{}).Where("email = ?", email).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("邮箱已存在")
	}

	// 哈希密码
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	user := model.User{
		Username: username,
		Email:    email,
		Password: string(hashed),
		Nickname: nickname,
		Status:   "active",
	}

	if err := l.db.Create(&user).Error; err != nil {
		return nil, fmt.Errorf("创建用户失败: %w", err)
	}

	l.events.EmitAsync("user.created", core.UserEvent{UserID: user.ID})
	return &user, nil
}

// GetByID 按 ID 获取用户
// slave 模式下优先从 Context 获取，如果 Context 中无用户信息或ID不匹配，则返回错误
func (l *UserLogic) GetByID(ctx context.Context, id int64) (*model.User, error) {
	// slave 模式：优先从 Context 获取用户信息
	if l.mode == "slave" {
		userInfo := core.GetUserFromCtx(ctx)
		if userInfo != nil && userInfo.ID == id {
			// Context 中有匹配的用户信息，直接返回
			return &model.User{
				ID:       userInfo.ID,
				Username: userInfo.Username,
				Email:    userInfo.Email,
			}, nil
		}
		// Context 中无用户信息或ID不匹配，返回 403 错误
		return nil, &SlaveModeError{Message: "SSO从属模式下无法查询其他用户信息，请联系SSO系统管理员"}
	}

	// master 模式：从本地数据库查询
	var user model.User
	if err := l.db.First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return &user, nil
}

// Update 更新用户信息（管理员）
// 仅在 master 模式下可用
func (l *UserLogic) Update(id int64, username, email, nickname, status string) error {
	// slave 模式下禁止修改
	if l.mode == "slave" {
		return &SlaveModeError{Message: "SSO从属模式下禁止修改用户信息，请使用SSO系统管理用户"}
	}

	updates := map[string]interface{}{}
	if username != "" {
		updates["username"] = username
	}
	if email != "" {
		updates["email"] = email
	}
	if nickname != "" {
		updates["nickname"] = nickname
	}
	if status != "" {
		updates["status"] = status
	}

	result := l.db.Model(&model.User{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return fmt.Errorf("更新失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在")
	}

	l.events.EmitAsync("user.updated", core.UserEvent{UserID: id})
	return nil
}

// Delete 删除用户（软删除）
// 仅在 master 模式下可用
func (l *UserLogic) Delete(id int64) error {
	// slave 模式下禁止删除
	if l.mode == "slave" {
		return &SlaveModeError{Message: "SSO从属模式下禁止删除用户，请使用SSO系统管理用户"}
	}

	result := l.db.Delete(&model.User{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("用户不存在")
	}

	l.events.EmitAsync("user.deleted", core.UserEvent{UserID: id})
	return nil
}

// ---------------------------------------------------------------------------
// 初始化
// ---------------------------------------------------------------------------

// InitAdmin 初始化默认用户账号（首次启动时）
// 仅在 master 模式下执行，slave 模式下跳过
func (l *UserLogic) InitAdmin() error {
	// slave 模式：不创建本地管理员账号
	if l.mode == "slave" {
		return nil
	}

	var count int64
	l.db.Model(&model.User{}).Count(&count)
	if count > 0 {
		return nil // 已有用户，跳过
	}

	// 种子用户列表：username / email / password / nickname
	seedUsers := []struct {
		Username string
		Email    string
		Password string
		Nickname string
	}{
		{"admin", "admin@gocms.local", "admin123", "管理员"},
		{"editor", "editor@gocms.local", "editor123", "编辑"},
		{"author", "author@gocms.local", "author123", "作者"},
	}

	for _, su := range seedUsers {
		hashed, err := bcrypt.GenerateFromPassword([]byte(su.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("密码加密失败(%s): %w", su.Username, err)
		}
		user := model.User{
			Username: su.Username,
			Email:    su.Email,
			Password: string(hashed),
			Nickname: su.Nickname,
			Status:   "active",
		}
		if err := l.db.Create(&user).Error; err != nil {
			return fmt.Errorf("创建用户 %s 失败: %w", su.Username, err)
		}
	}

	return nil
}
