package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAppProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppProfileLogic {
	return &GetAppProfileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAppProfileLogic) GetAppProfile(in *pb.AppAccessRequest) (*pb.AppUserInfo, error) {
	return profile(l.ctx, l.svcCtx, in)
}
