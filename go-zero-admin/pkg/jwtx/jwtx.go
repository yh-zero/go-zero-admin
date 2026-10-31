package jwtx

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// CustomClaims 后台用户 JWT claims
type CustomClaims struct {
	UserID    int64  `json:"user_id"`
	Username  string `json:"username"`
	DeptId    int64  `json:"dept_id"`
	SessionId string `json:"sid"`
	jwt.RegisteredClaims
}

// GenerateAccessToken 生成后台访问令牌
func GenerateAccessToken(secret string, userID int64, username string, deptID int64, expireSeconds int64) (string, error) {
	accessId, err := RandomToken()
	if err != nil {
		return "", err
	}
	return GenerateSessionAccessToken(secret, &Session{UserId: userID, Username: username, DeptId: deptID, AccessId: accessId}, expireSeconds)
}

func GenerateSessionAccessToken(secret string, session *Session, expireSeconds int64) (string, error) {
	claims := CustomClaims{
		UserID:    session.UserId,
		Username:  session.Username,
		DeptId:    session.DeptId,
		SessionId: session.Id,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        session.AccessId,
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
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
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
// App 端 JWT（独立 audience / issuer，权限以服务端当前状态为准）
// ============================================

// AppClaims 只作为会话凭证；状态和会员字段不能代替当前数据库校验。
type AppClaims struct {
	UserID    int64  `json:"uid"`
	SessionId string `json:"sid"`
	Status    string `json:"sta"` // active / pending / disabled / rejected
	Vip       bool   `json:"vip"` // 有效会员快照（登录/刷新时计算）
	VipPlan   string `json:"plan,omitempty"`
	jwt.RegisteredClaims
}

// GenerateAppToken 生成 App 访问令牌（带权限快照）
func GenerateAppToken(secret string, userID int64, status string, vip bool, vipPlan string, expireSeconds int64) (string, error) {
	accessId, err := RandomToken()
	if err != nil {
		return "", err
	}
	return GenerateAppSessionToken(secret, &Session{UserId: userID, AccessId: accessId}, status, vip, vipPlan, expireSeconds)
}

func GenerateAppSessionToken(secret string, session *Session, status string, vip bool, vipPlan string, expireSeconds int64) (string, error) {
	claims := AppClaims{
		UserID:    session.UserId,
		SessionId: session.Id,
		Status:    status,
		Vip:       vip,
		VipPlan:   vipPlan,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expireSeconds) * time.Second)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ID:        session.AccessId,
			Issuer:    "go-zero-app",
			Audience:  jwt.ClaimStrings{"go-zero-app"},
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
		}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired(), jwt.WithIssuer("go-zero-app"), jwt.WithAudience("go-zero-app"))
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AppClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
