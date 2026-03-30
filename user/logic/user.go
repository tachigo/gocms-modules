// Package logic user 业务逻辑
// 用户认证、个人信息管理、用户 CRUD
package logic

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"gocms/internal/core"
	"gocms/internal/module/user/model"
)

// UserLogic 用户业务逻辑
type UserLogic struct {
	db     *gorm.DB
	jwt    *JWTManager
	events core.EventBus
}

// NewUserLogic 创建用户逻辑实例
func NewUserLogic(db *gorm.DB, jwt *JWTManager, events core.EventBus) *UserLogic {
	return &UserLogic{db: db, jwt: jwt, events: events}
}

// JWTManager 获取 JWT 管理器（供中间件使用）
func (l *UserLogic) JWTManager() *JWTManager {
	return l.jwt
}

// ---------------------------------------------------------------------------
// 认证相关
// ---------------------------------------------------------------------------

// Login 用户登录，验证成功返回 JWT Token
func (l *UserLogic) Login(username, password string) (string, *model.User, error) {
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
func (l *UserLogic) GetProfile(userID int64) (*model.User, error) {
	var user model.User
	if err := l.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return &user, nil
}

// UpdateProfile 更新个人信息（昵称、头像）
func (l *UserLogic) UpdateProfile(userID int64, nickname, avatar string) error {
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
func (l *UserLogic) ChangePassword(userID int64, oldPassword, newPassword string) error {
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
func (l *UserLogic) List(page, pageSize int) ([]model.User, int64, error) {
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
func (l *UserLogic) Create(username, email, password, nickname string) (*model.User, error) {
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
func (l *UserLogic) GetByID(id int64) (*model.User, error) {
	var user model.User
	if err := l.db.First(&user, id).Error; err != nil {
		return nil, fmt.Errorf("用户不存在")
	}
	return &user, nil
}

// Update 更新用户信息（管理员）
func (l *UserLogic) Update(id int64, username, email, nickname, status string) error {
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
func (l *UserLogic) Delete(id int64) error {
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
// 创建 admin（管理员）、editor（编辑）、author（作者）三个测试账号
func (l *UserLogic) InitAdmin() error {
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
