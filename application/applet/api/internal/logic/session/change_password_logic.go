package session

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
)

type ChangePasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordRequest) (*types.MessageResponse, error) {
	_, err := l.svcCtx.AppletUserRPC.ChangePassword(l.ctx, &pb.ChangePasswordRequest{Session: sessionRequest(l.ctx), OldPassword: req.OldPassword, NewPassword: req.NewPassword})
	return &types.MessageResponse{Message: "密码已修改，请重新登录"}, err
}
