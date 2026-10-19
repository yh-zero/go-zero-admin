// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appmembers

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppUserSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询 App 用户有效设备
func NewListAppUserSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppUserSessionsLogic {
	return &ListAppUserSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppUserSessionsLogic) ListAppUserSessions(req *types.AppUserIDRequest) (resp *types.AppSessionList, err error) {
	reply, err := l.svcCtx.AppAccountRPC.ListAppUserSessions(l.ctx, &pb.AppAdminUserRequest{Actor: appbridge.Actor(l.ctx), UserID: req.UserID})
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
