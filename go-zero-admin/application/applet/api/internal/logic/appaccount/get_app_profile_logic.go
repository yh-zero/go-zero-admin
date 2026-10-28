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

type GetAppProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 读取本人资料
func NewGetAppProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppProfileLogic {
	return &GetAppProfileLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAppProfileLogic) GetAppProfile() (resp *types.AppUser, err error) {
	reply, err := l.svcCtx.AppAccountRPC.GetAppProfile(l.ctx, &pb.AppAccessRequest{AccessToken: appbridge.Request(l.ctx).Token})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppUser](reply)
}
