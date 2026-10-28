package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAppProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppProfileLogic {
	return &UpdateAppProfileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateAppProfileLogic) UpdateAppProfile(in *pb.AppProfileRequest) (*pb.AppUserInfo, error) {
	return updateProfile(l.ctx, l.svcCtx, in)
}
