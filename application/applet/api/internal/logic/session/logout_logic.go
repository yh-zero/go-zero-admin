package session

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *LogoutLogic) Logout(req *types.LogoutRequest) (*types.MessageResponse, error) {
	_, err := l.svcCtx.AppletUserRPC.Logout(l.ctx, sessionRequest(l.ctx))
	return &types.MessageResponse{Message: "已退出登录"}, err
}
