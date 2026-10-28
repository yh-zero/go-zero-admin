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

type RevokeAppUserSessionLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销 App 用户设备
func NewRevokeAppUserSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeAppUserSessionLogic {
	return &RevokeAppUserSessionLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeAppUserSessionLogic) RevokeAppUserSession(req *types.AppSessionIDRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppAccountRPC.RevokeAppUserSession(l.ctx, &pb.AppAdminRevokeRequest{Actor: appbridge.Actor(l.ctx), ID: req.ID})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "操作成功"}, nil
}
