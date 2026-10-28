// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package devicesession

import (
	"context"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyDeviceSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询自己的有效设备会话
func NewGetMyDeviceSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyDeviceSessionsLogic {
	return &GetMyDeviceSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyDeviceSessionsLogic) GetMyDeviceSessions(req *types.DeviceSessionListRequest) (resp *types.DeviceSessionListResponse, err error) {
	if req.PageNo == 0 {
		req.PageNo = l.svcCtx.Config.Page.PageNo
	}
	if req.PageSize == 0 {
		req.PageSize = l.svcCtx.Config.Page.PageSize
	}
	current := actor(l.ctx)
	response, err := l.svcCtx.AppletSessionManageRPC.GetDeviceSessions(l.ctx, &pb.DeviceSessionListRequest{Actor: current, UserID: current.UserID, PageRequest: &pb.PageRequest{PageNo: req.PageNo, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	return deviceResponse(response, req.PageNo, req.PageSize), nil
}
