// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginAppAccountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 已审核 App 账号登录
func NewLoginAppAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginAppAccountLogic {
	return &LoginAppAccountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LoginAppAccountLogic) LoginAppAccount(req *types.AppLoginRequest) (resp *types.AppAuthTokens, err error) {
	reply, err := l.svcCtx.AppAccountRPC.LoginAppAccount(l.ctx, &pb.AppLoginRequest{Account: req.Account, Password: req.Password, IP: appbridge.Request(l.ctx).IP, UserAgent: appbridge.Request(l.ctx).UserAgent})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppAuthTokens](reply)
}
