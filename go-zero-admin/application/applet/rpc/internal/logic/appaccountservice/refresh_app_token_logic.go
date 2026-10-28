package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshAppTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefreshAppTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshAppTokenLogic {
	return &RefreshAppTokenLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RefreshAppTokenLogic) RefreshAppToken(in *pb.AppRefreshRequest) (*pb.AppAuthTokens, error) {
	return refresh(l.ctx, l.svcCtx, in)
}
