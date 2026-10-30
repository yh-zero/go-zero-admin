package authlogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	rpcResp, err := l.svcCtx.AuthRpc.GetCaptcha(l.ctx, &adminpb.GetCaptchaRequest{})
	if err != nil {
		return nil, err
	}
	return &types.GetCaptchaResp{
		CaptchaID:   rpcResp.Captcha.CaptchaID,
		ImageBase64: rpcResp.Captcha.ImageBase64,
		ExpiresAt:   rpcResp.Captcha.ExpiresAt,
	}, nil
}
