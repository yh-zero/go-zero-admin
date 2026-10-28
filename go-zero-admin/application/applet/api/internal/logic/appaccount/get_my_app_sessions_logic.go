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

type GetMyAppSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查看本人有效设备
func NewGetMyAppSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyAppSessionsLogic {
	return &GetMyAppSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyAppSessionsLogic) GetMyAppSessions() (resp *types.AppSessionList, err error) {
	reply, err := l.svcCtx.AppAccountRPC.GetMyAppSessions(l.ctx, &pb.AppAccessRequest{AccessToken: appbridge.Request(l.ctx).Token})
	if err != nil {
		return nil, err
	}
	response, err := appbridge.Copy[types.AppSessionList](reply)
	if err != nil {
		return nil, err
	}
	if response.List == nil {
		response.List = []types.AppSession{}
	}
	return response, nil
}
