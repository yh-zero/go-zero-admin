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

type RevokeDeviceSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 管理员撤销一个设备会话
func NewRevokeDeviceSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDeviceSessionLogic {
	return &RevokeDeviceSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeDeviceSessionLogic) RevokeDeviceSession(req *types.RevokeDeviceSessionRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletSessionManageRPC.RevokeDeviceSession(l.ctx, &pb.RevokeDeviceSessionRequest{Actor: actor(l.ctx), ID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "设备会话已撤销"}, nil
}
