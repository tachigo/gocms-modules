// Package logic JWT Token 工具
// 生成 / 验证 / 黑名单管理
package logic

import (
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims JWT 载荷
type Claims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// JWTManager JWT 管理器
type JWTManager struct {
	secret []byte
	expire time.Duration
	issuer string

	// Token 级别黑名单（登出时使用）
	blacklist map[string]time.Time
	// 用户级别失效时间戳（改密码后使该用户所有旧 token 失效）
	userInvalidatedAt map[int64]time.Time
	mu                sync.RWMutex
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secret string, expireHours int, issuer string) *JWTManager {
	return &JWTManager{
		secret:            []byte(secret),
		expire:            time.Duration(expireHours) * time.Hour,
		issuer:            issuer,
		blacklist:         make(map[string]time.Time),
		userInvalidatedAt: make(map[int64]time.Time),
	}
}

// GenerateToken 生成 JWT Token
func (m *JWTManager) GenerateToken(userID int64, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.expire)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseToken 解析并验证 JWT Token
// 同时检查黑名单和用户级别失效
func (m *JWTManager) ParseToken(tokenString string) (*Claims, error) {
	// 1. 检查 Token 黑名单
	if m.IsBlacklisted(tokenString) {
		return nil, fmt.Errorf("token已失效")
	}

	// 2. 解析 Token
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("token无效: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("token无效")
	}

	// 3. 检查用户级别失效（改密码后签发的 token 才有效）
	m.mu.RLock()
	invalidatedAt, exists := m.userInvalidatedAt[claims.UserID]
	m.mu.RUnlock()
	if exists && claims.IssuedAt != nil && claims.IssuedAt.Time.Before(invalidatedAt) {
		return nil, fmt.Errorf("token已失效（密码已修改）")
	}

	return claims, nil
}

// AddToBlacklist 将单个 Token 加入黑名单（登出）
func (m *JWTManager) AddToBlacklist(tokenString string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklist[tokenString] = time.Now().Add(m.expire)
}

// IsBlacklisted 检查 Token 是否在黑名单中
func (m *JWTManager) IsBlacklisted(tokenString string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.blacklist[tokenString]
	return exists
}

// InvalidateUser 使某用户的所有旧 Token 失效（改密码后调用）
func (m *JWTManager) InvalidateUser(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.userInvalidatedAt[userID] = time.Now()
}

// CleanExpired 清理已过期的黑名单条目（定期调用，减少内存占用）
func (m *JWTManager) CleanExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for token, expireAt := range m.blacklist {
		if now.After(expireAt) {
			delete(m.blacklist, token)
		}
	}
}
