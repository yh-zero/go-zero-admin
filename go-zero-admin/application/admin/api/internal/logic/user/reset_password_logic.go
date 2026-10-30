package user

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ResetPasswordLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ResetPasswordLogic) ResetPassword(req *types.ResetPasswordReq) error {
	_, err := l.svcCtx.UserRpc.ResetPassword(l.ctx, &adminpb.ResetPasswordRequest{
		UserId: req.Id, NewPassword: req.NewPassword,
	})
	if err != nil {
		return err
	}
	return nil
}
