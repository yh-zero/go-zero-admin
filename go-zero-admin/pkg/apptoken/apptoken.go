// Package apptoken App 端统一鉴权与权益判断
// 纯函数式接口：依赖（密钥/Redis/DB）显式传入，不绑定具体模块
package apptoken

import (
	"context"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
)

// 权益缓存 key 前缀
const (
	vipKeyFmt     = "app:user:%d:vip"
	benefitPrefix = "app:user:%d:benefit:"
	benefitTTL    = 300 * time.Second
)

// Claims 从 token 解析 AppClaims（含 status/vip 快照）
func Claims(secret, token string) (*jwtx.AppClaims, error) {
	claims, err := jwtx.ParseAppToken(secret, token)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	return claims, nil
}

// MustActive 校验用户状态为 active
func MustActive(claims *jwtx.AppClaims) error {
	if claims.Status != "active" {
		return xerr.NewErrCodeMsg(400, "账号未激活或已停用")
	}
	return nil
}

// IsVip 判断有效会员（JWT 快照优先，Redis 缓存兜底）
func IsVip(ctx context.Context, rdb *redis.Redis, claims *jwtx.AppClaims) bool {
	if claims.Vip {
		return true
	}
	key := fmt.Sprintf(vipKeyFmt, claims.UserID)
	val, err := rdb.GetCtx(ctx, key)
	return err == nil && val == "1"
}

// HasBenefit 判断具体权益（Redis 缓存，未命中回源 DB）
func HasBenefit(ctx context.Context, rdb *redis.Redis, db sqlx.SqlConn, userID int64, code string) (bool, error) {
	key := fmt.Sprintf(benefitPrefix+"%s", userID, code)
	val, err := rdb.GetCtx(ctx, key)
	if err == nil && val == "1" {
		return true, nil
	}

	var cnt int
	err = db.QueryRowCtx(ctx, &cnt,
		"SELECT COUNT(*) FROM app_membership_grant WHERE user_id=? AND revoked=0 AND benefits LIKE ? AND starts_at<=NOW() AND expires_at>=NOW()",
		userID, "%"+code+"%")
	if err != nil {
		return false, err
	}
	if cnt > 0 {
		_ = rdb.SetexCtx(ctx, key, "1", int(benefitTTL.Seconds()))
		return true, nil
	}
	return false, nil
}

// RefreshVipCache 登录时计算有效会员并刷新缓存
func RefreshVipCache(ctx context.Context, rdb *redis.Redis, db sqlx.SqlConn, userID int64) (bool, string) {
	var planCode string
	err := db.QueryRowCtx(ctx, &planCode,
		"SELECT plan_code FROM app_membership_grant WHERE user_id=? AND revoked=0 AND starts_at<=NOW() AND expires_at>=NOW() ORDER BY expires_at DESC LIMIT 1",
		userID)
	if err != nil {
		DelVipCache(ctx, rdb, userID)
		return false, ""
	}
	key := fmt.Sprintf(vipKeyFmt, userID)
	_ = rdb.SetexCtx(ctx, key, "1", int((30 * 24 * time.Hour).Seconds()))
	return true, planCode
}

// DelVipCache 授予/撤销会员时主动失效缓存
func DelVipCache(ctx context.Context, rdb *redis.Redis, userID int64) {
	_, _ = rdb.DelCtx(ctx, fmt.Sprintf(vipKeyFmt, userID))
}
