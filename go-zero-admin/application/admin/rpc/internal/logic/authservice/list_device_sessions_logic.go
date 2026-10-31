package authservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ListDeviceSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListDeviceSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListDeviceSessionsLogic {
	return &ListDeviceSessionsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListDeviceSessionsLogic) ListDeviceSessions(in *adminpb.ListDeviceSessionsRequest) (*adminpb.ListDeviceSessionsResponse, error) {
	if in.Actor == nil || in.Actor.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	sessions, err := l.svcCtx.Sessions.List(l.ctx, in.Actor.UserId)
	if err != nil {
		return nil, sessionError(err)
	}
	list := make([]*adminpb.DeviceSessionInfo, 0, len(sessions))
	for _, session := range sessions {
		list = append(list, &adminpb.DeviceSessionInfo{Id: session.Id, UserId: session.UserId, DeviceName: session.DeviceName, Ip: session.Ip, UserAgent: session.UserAgent, CreatedAt: session.CreatedAt, LastActiveAt: session.LastSeenAt, ExpiresAt: session.ExpiresAt, Current: session.Id == in.CurrentSessionId})
	}
	return &adminpb.ListDeviceSessionsResponse{List: list, Page: &adminpb.PageResponse{Total: int64(len(list))}}, nil
}
