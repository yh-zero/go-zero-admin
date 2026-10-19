package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type LogoutAppAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutAppAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutAppAccountLogic {
	return &LogoutAppAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *LogoutAppAccountLogic) LogoutAppAccount(in *pb.AppLogoutRequest) (*pb.NoDataResponse, error) {
	return logout(l.ctx, l.svcCtx, in)
}
