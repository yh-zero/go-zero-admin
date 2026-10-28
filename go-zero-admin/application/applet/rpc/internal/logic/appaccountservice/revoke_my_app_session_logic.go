package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RevokeMyAppSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeMyAppSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeMyAppSessionLogic {
	return &RevokeMyAppSessionLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RevokeMyAppSessionLogic) RevokeMyAppSession(in *pb.AppRevokeSelfRequest) (*pb.NoDataResponse, error) {
	return revokeMine(l.ctx, l.svcCtx, in)
}
