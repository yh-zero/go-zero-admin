package user

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ChangePasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ChangePasswordLogic) ChangePassword(req *types.ChangePasswordReq) error {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	_, err := l.svcCtx.UserRpc.ChangePassword(l.ctx, &adminpb.ChangePasswordRequest{
		Actor:       &adminpb.Actor{UserId: userID},
		OldPassword: req.OldPassword, NewPassword: req.NewPassword,
	})
	if err != nil {
		return err
	}
	return nil
}
