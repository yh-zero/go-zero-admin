package auth

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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

func (l *GetCaptchaLogic) GetCaptcha(req *types.GetCaptchaReq) (resp *types.GetCaptchaResp, err error) {
	ip, _ := l.ctx.Value(middleware.CtxKeyClientIP).(string)
	const script = `local n=redis.call('INCR',KEYS[1]);if n==1 then redis.call('EXPIRE',KEYS[1],60) end;return n`
	count, err := l.svcCtx.Redis.Eval(l.ctx, script, []string{"admin:captcha:ip:" + ip}).Int()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if count > 60 {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "验证码请求过于频繁，请稍后再试")
	}
	rpcResp, err := l.svcCtx.AuthRpc.GetCaptcha(l.ctx, &adminpb.GetCaptchaRequest{})
	if err != nil {
		return nil, err
	}
	return &types.GetCaptchaResp{
		// 兼容尚未升级的 RPC：旧响应没有 Enabled 字段，但仍返回验证码。
		Enabled:     rpcResp.Captcha.Enabled || rpcResp.Captcha.CaptchaID != "" || rpcResp.Captcha.ImageBase64 != "",
		CaptchaID:   rpcResp.Captcha.CaptchaID,
		ImageBase64: rpcResp.Captcha.ImageBase64,
		ExpiresAt:   rpcResp.Captcha.ExpiresAt,
	}, nil
}
