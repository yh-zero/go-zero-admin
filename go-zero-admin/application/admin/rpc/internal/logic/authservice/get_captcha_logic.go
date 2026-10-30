package authservicelogic

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/redis"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetCaptchaLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetCaptchaLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetCaptchaLogic {
	return &GetCaptchaLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// GetCaptcha 生成图片验证码（简版：返回文本验证码，生产可换图片库）
func (l *GetCaptchaLogic) GetCaptcha(in *adminpb.GetCaptchaRequest) (*adminpb.GetCaptchaResponse, error) {
	// 生成 4 位数字验证码
	code, err := l.generateCode(4)
	if err != nil {
		l.Errorf("generate captcha error: %v", err)
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	// 生成唯一 ID
	id, err := l.generateCode(16)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	// 存 Redis，120 秒过期
	rdb := redis.MustNewRedis(redis.RedisConf{
		Host: l.svcCtx.Config.CacheRedis[0].Host,
		Type: l.svcCtx.Config.CacheRedis[0].Type,
	})
	key := "captcha:" + id
	expire := l.svcCtx.Config.Captcha.ExpireSeconds
	if expire <= 0 {
		expire = 120
	}
	err = rdb.Setex(key, code, expire)
	if err != nil {
		l.Errorf("save captcha error: %v", err)
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	// 简版：直接返回文本，生产环境换成 base64 图片
	imageBase64 := base64.StdEncoding.EncodeToString([]byte(code))

	return &adminpb.GetCaptchaResponse{
		Captcha: &adminpb.CaptchaInfo{
			CaptchaID:   id,
			ImageBase64: imageBase64,
			ExpiresAt:   time.Now().Unix() + int64(expire),
		},
	}, nil
}

// generateCode 生成指定长度的随机数字字符串
func (l *GetCaptchaLogic) generateCode(length int) (string, error) {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	code := ""
	for _, v := range b {
		code += fmt.Sprintf("%d", int(v)%10)
	}
	return code, nil
}
