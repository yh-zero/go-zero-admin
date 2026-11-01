package appadminservicelogic

import (
	"context"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppSessionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppSessionsLogic {
	return &ListAppSessionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *ListAppSessionsLogic) ListAppSessions(in *apppb.ListAppSessionsRequest) (*apppb.ListAppSessionsResponse, error) {
	if err := requireActor(in.GetActor()); err != nil {
		return nil, err
	}
	if _, err := liveAppUser(l.ctx, l.svcCtx, in.UserId); err != nil {
		return nil, err
	}
	sessions, err := l.svcCtx.Sessions.List(l.ctx, in.UserId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	list := make([]*apppb.AppDeviceSession, 0, len(sessions))
	for _, session := range sessions {
		list = append(list, &apppb.AppDeviceSession{
			Id: session.Id, DeviceName: session.DeviceName, Ip: session.Ip,
			UserAgent: session.UserAgent, CreatedAt: session.CreatedAt,
			LastSeenAt: session.LastSeenAt, ExpiresAt: session.ExpiresAt,
		})
	}
	return &apppb.ListAppSessionsResponse{List: list}, nil
}
