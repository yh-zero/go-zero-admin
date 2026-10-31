// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package auth

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListDeviceSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListDeviceSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDeviceSessionsLogic {
	return &ListDeviceSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListDeviceSessionsLogic) ListDeviceSessions(req *types.ListDeviceSessionsReq) (resp *types.ListDeviceSessionsResp, err error) {
	userId, _ := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	sessionId, _ := l.ctx.Value(middleware.CtxKeySessionId).(string)
	rpcResp, err := l.svcCtx.AuthRpc.ListDeviceSessions(l.ctx, &adminpb.ListDeviceSessionsRequest{Actor: &adminpb.Actor{UserId: userId}, CurrentSessionId: sessionId})
	if err != nil {
		return nil, err
	}
	list := make([]types.DeviceSession, 0, len(rpcResp.List))
	for _, item := range rpcResp.List {
		list = append(list, types.DeviceSession{Id: item.Id, DeviceName: item.DeviceName, Ip: item.Ip, UserAgent: item.UserAgent, CreatedAt: item.CreatedAt, LastSeenAt: item.LastActiveAt, ExpiresAt: item.ExpiresAt, IsCurrent: item.Current})
	}
	return &types.ListDeviceSessionsResp{List: list}, nil
}
