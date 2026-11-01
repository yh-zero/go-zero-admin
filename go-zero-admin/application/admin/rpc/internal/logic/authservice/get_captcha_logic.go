package authservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/jwtx"
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

// GetCaptcha 生成一次性图片验证码。
func (l *GetCaptchaLogic) GetCaptcha(in *adminpb.GetCaptchaRequest) (*adminpb.GetCaptchaResponse, error) {
	if !l.svcCtx.Config.Captcha.Enabled {
		return &adminpb.GetCaptchaResponse{Captcha: &adminpb.CaptchaInfo{Enabled: false}}, nil
	}
	// 生成 4 位数字验证码
	code, err := randomDigits(4)
	if err != nil {
		l.Errorf("generate captcha error: %v", err)
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	// 生成唯一 ID
	id, err := jwtx.RandomToken()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	imageBase64, err := captchaImage(code)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	// 先完成图片绘制，再保存一次性答案。
	key := "captcha:" + id
	expire := l.svcCtx.Config.Captcha.ExpireSeconds
	if expire <= 0 {
		expire = 120
	}
	err = l.svcCtx.Redis.SetexCtx(l.ctx, key, code, expire)
	if err != nil {
		l.Errorf("save captcha error: %v", err)
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	return &adminpb.GetCaptchaResponse{
		Captcha: &adminpb.CaptchaInfo{
			Enabled:     true,
			CaptchaID:   id,
			ImageBase64: imageBase64,
			ExpiresAt:   time.Now().Unix() + int64(expire),
		},
	}, nil
}
