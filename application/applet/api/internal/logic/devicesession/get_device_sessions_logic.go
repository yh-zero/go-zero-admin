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

type GetDeviceSessionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 管理员查询有效设备会话
func NewGetDeviceSessionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeviceSessionsLogic {
	return &GetDeviceSessionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDeviceSessionsLogic) GetDeviceSessions(req *types.AdminDeviceSessionListRequest) (resp *types.DeviceSessionListResponse, err error) {
	if req.PageNo == 0 {
		req.PageNo = l.svcCtx.Config.Page.PageNo
	}
	if req.PageSize == 0 {
		req.PageSize = l.svcCtx.Config.Page.PageSize
	}
	response, err := l.svcCtx.AppletSessionManageRPC.GetDeviceSessions(l.ctx, &pb.DeviceSessionListRequest{Actor: actor(l.ctx), UserID: req.UserId, PageRequest: &pb.PageRequest{PageNo: req.PageNo, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	return deviceResponse(response, req.PageNo, req.PageSize), nil
}
