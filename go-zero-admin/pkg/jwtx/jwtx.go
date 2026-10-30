package jwtx

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/zeromicro/go-zero/core/timex"
)

// CustomClaims 后台用户 JWT claims
type CustomClaims struct {
	UserID   int64  `json:"user_id"`
	Username string `json:"username"`
	DeptId   int64  `json:"dept_id"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成后台访问令牌
func GenerateAccessToken(secret string, userID int64, username string, deptID int64, expireSeconds int64) (string, error) {
	claims := CustomClaims{
		UserID:   userID,
		Username: username,
		DeptId:   deptID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        timex.Now().String(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAccessToken 解析并验证后台访问令牌
func ParseAccessToken(secret, tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// ============================================
// App 端 JWT（带权限快照：status/vip）
// ============================================

// AppClaims App 用户 JWT claims（含权限快照，敏感接口零查库）
type AppClaims struct {
	UserID  int64  `json:"uid"`
	Status  string `json:"sta"` // active / pending / disabled / rejected
	Vip     bool   `json:"vip"` // 有效会员快照（登录/刷新时计算）
	VipPlan string `json:"plan,omitempty"`
	jwt.RegisteredClaims
}

// GenerateAppToken 生成 App 访问令牌（带权限快照）
func GenerateAppToken(secret string, userID int64, status string, vip bool, vipPlan string, expireSeconds int64) (string, error) {
	claims := AppClaims{
		UserID:  userID,
		Status:  status,
		Vip:     vip,
		VipPlan: vipPlan,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        generateJTI(),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ParseAppToken 解析 App 令牌
func ParseAppToken(secret, tokenString string) (*AppClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &AppClaims{},
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AppClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

// generateJTI 生成简单唯一 ID
func generateJTI() string {
	return time.Now().Format("20060102150405.000000000")
}
