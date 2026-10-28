package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetAppUserStatusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetAppUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAppUserStatusLogic {
	return &SetAppUserStatusLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SetAppUserStatusLogic) SetAppUserStatus(in *pb.AppStatusRequest) (*pb.NoDataResponse, error) {
	return statusUser(l.ctx, l.svcCtx, in)
}
