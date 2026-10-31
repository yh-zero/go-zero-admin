package userservicelogic

import (
	"context"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ResetPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ResetPasswordLogic) ResetPassword(in *adminpb.ResetPasswordRequest) (*adminpb.NoDataResponse, error) {
	if in.UserId <= 0 || in.GetActor().GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	actor := model.UserActor{UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId()}
	user, err := l.svcCtx.SysUserModel.FindScopedUser(l.ctx, in.UserId, actor)
	if err != nil {
		return nil, passwordMutationError(err)
	}
	if user.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	if user.Status != 1 {
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), 10)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	err = l.svcCtx.SysUserModel.UpdatePassword(l.ctx, user.Id, actor, "", string(hash), func(ctx context.Context, userId int64) error {
		if err := l.svcCtx.Sessions.RevokeAll(ctx, userId); err != nil {
			return xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		return nil
	})
	if err != nil {
		return nil, passwordMutationError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}
