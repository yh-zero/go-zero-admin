package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type LoginAppAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginAppAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginAppAccountLogic {
	return &LoginAppAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LoginAppAccountLogic) LoginAppAccount(in *pb.AppLoginRequest) (*pb.AppAuthTokens, error) {
	return login(l.ctx, l.svcCtx, in)
}
