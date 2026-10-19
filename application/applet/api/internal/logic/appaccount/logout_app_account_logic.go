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

type LogoutAppAccountLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 退出当前或全部设备
func NewLogoutAppAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutAppAccountLogic {
	return &LogoutAppAccountLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *LogoutAppAccountLogic) LogoutAppAccount(req *types.AppLogoutRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppAccountRPC.LogoutAppAccount(l.ctx, &pb.AppLogoutRequest{AccessToken: appbridge.Request(l.ctx).Token, AllDevices: req.AllDevices})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "操作成功"}, nil
}
