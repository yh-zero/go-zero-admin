// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appaccount

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新本人昵称
func NewUpdateAppProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppProfileLogic {
	return &UpdateAppProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateAppProfileLogic) UpdateAppProfile(req *types.AppProfileRequest) (resp *types.AppUser, err error) {
	reply, err := l.svcCtx.AppAccountRPC.UpdateAppProfile(l.ctx, &pb.AppProfileRequest{AccessToken: appbridge.Request(l.ctx).Token, Nickname: req.Nickname})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppUser](reply)
}
