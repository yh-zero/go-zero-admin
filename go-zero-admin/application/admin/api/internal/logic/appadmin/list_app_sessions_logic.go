// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appadmin

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ListAppSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListAppSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppSessionsLogic {
	return &ListAppSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListAppSessionsLogic) ListAppSessions(req *types.AppListSessionsReq) (resp *types.ListDeviceSessionsResp, err error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	rpcResp, err := l.svcCtx.AppAdminRpc.ListAppSessions(l.ctx, &apppb.ListAppSessionsRequest{
		Actor: appAdminActor(l.ctx), UserId: req.Id,
	})
	if err != nil {
		return nil, err
	}
	list := make([]types.DeviceSession, 0, len(rpcResp.List))
	for _, session := range rpcResp.List {
		list = append(list, types.DeviceSession{
			Id: session.Id, DeviceName: session.DeviceName, Ip: session.Ip,
			UserAgent: session.UserAgent, CreatedAt: session.CreatedAt,
			LastSeenAt: session.LastSeenAt, ExpiresAt: session.ExpiresAt, IsCurrent: false,
		})
	}
	return &types.ListDeviceSessionsResp{List: list}, nil
}
