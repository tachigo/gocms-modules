// Package logic JWT Token 工具
package logic

import (
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gocms/internal/core"
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
	cache  core.CacheEngine // 使用 CacheEngine 持久化黑名单
}

// NewJWTManager 创建 JWT 管理器
func NewJWTManager(secret string, expireHours int, issuer string, cache core.CacheEngine) *JWTManager {
	return &JWTManager{
		secret: []byte(secret),
		expire: time.Duration(expireHours) * time.Hour,
		issuer: issuer,
		cache:  cache,
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
	if m.isUserInvalidated(claims.UserID, claims.IssuedAt.Time) {
		return nil, fmt.Errorf("token已失效（密码已修改）")
	}

	return claims, nil
}

// AddToBlacklist 将单个 Token 加入黑名单（登出）
func (m *JWTManager) AddToBlacklist(tokenString string) {
	key := "jwt:blacklist:" + tokenString
	m.cache.Set(key, []byte("1"), m.expire)
}

// IsBlacklisted 检查 Token 是否在黑名单中
func (m *JWTManager) IsBlacklisted(tokenString string) bool {
	key := "jwt:blacklist:" + tokenString
	_, err := m.cache.Get(key)
	return err == nil
}

// InvalidateUser 使某用户的所有旧 Token 失效（改密码后调用）
func (m *JWTManager) InvalidateUser(userID int64) {
	key := "jwt:user_invalid:" + strconv.FormatInt(userID, 10)
	m.cache.Set(key, []byte(strconv.FormatInt(time.Now().Unix(), 10)), m.expire)
}

// isUserInvalidated 检查用户是否已被标记为失效
func (m *JWTManager) isUserInvalidated(userID int64, issuedAt time.Time) bool {
	key := "jwt:user_invalid:" + strconv.FormatInt(userID, 10)
	data, err := m.cache.Get(key)
	if err != nil {
		return false
	}
	invalidatedAtUnix, _ := strconv.ParseInt(string(data), 10, 64)
	invalidatedAt := time.Unix(invalidatedAtUnix, 0)
	return issuedAt.Before(invalidatedAt)
}
