package authlogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	rpcResp, err := l.svcCtx.AuthRpc.Login(l.ctx, &adminpb.LoginRequest{
		Username:  req.Username,
		Password:  req.Password,
		CaptchaId: req.CaptchaID,
		Captcha:   req.Captcha,
		Ip:        "127.0.0.1",
		UserAgent: "test",
	})
	if err != nil {
		return nil, err
	}
	return &types.LoginResp{
		AccessToken: rpcResp.AccessToken,
		ExpiresAt:   rpcResp.ExpiresAt,
	}, nil
}
