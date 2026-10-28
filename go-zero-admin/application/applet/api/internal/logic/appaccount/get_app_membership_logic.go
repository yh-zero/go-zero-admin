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

type GetAppMembershipLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 读取当前会员权益
func NewGetAppMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppMembershipLogic {
	return &GetAppMembershipLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetAppMembershipLogic) GetAppMembership() (resp *types.AppMembership, err error) {
	reply, err := l.svcCtx.AppMembershipRPC.GetAppMembership(l.ctx, &pb.AppAccessRequest{AccessToken: appbridge.Request(l.ctx).Token})
	if err != nil {
		return nil, err
	}
	response, err := appbridge.Copy[types.AppMembership](reply)
	if err != nil {
		return nil, err
	}
	if response.Entitlements == nil {
		response.Entitlements = []string{}
	}
	return response, nil
}
