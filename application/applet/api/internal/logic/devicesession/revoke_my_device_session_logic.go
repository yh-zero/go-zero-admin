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

type RevokeMyDeviceSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销自己的一个设备会话
func NewRevokeMyDeviceSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeMyDeviceSessionLogic {
	return &RevokeMyDeviceSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeMyDeviceSessionLogic) RevokeMyDeviceSession(req *types.RevokeDeviceSessionRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletSessionManageRPC.RevokeDeviceSession(l.ctx, &pb.RevokeDeviceSessionRequest{Actor: actor(l.ctx), ID: req.Id, SelfOnly: true})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "设备会话已撤销"}, nil
}
