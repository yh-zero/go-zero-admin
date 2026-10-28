package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeAppUserSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeAppUserSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeAppUserSessionLogic {
	return &RevokeAppUserSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RevokeAppUserSessionLogic) RevokeAppUserSession(in *pb.AppAdminRevokeRequest) (*pb.NoDataResponse, error) {
	return revokeAdminSession(l.ctx, l.svcCtx, in)
}
