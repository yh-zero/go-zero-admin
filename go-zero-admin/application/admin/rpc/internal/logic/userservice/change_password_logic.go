package userservicelogic

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ChangePasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ChangePasswordLogic) ChangePassword(in *adminpb.ChangePasswordRequest) (*adminpb.NoDataResponse, error) {
	if err := validateNewPassword(in.NewPassword); err != nil {
		return nil, err
	}
	if in.GetActor().GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.GetActor().GetUserId()))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	if user.DeletedAt.Valid {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	if user.Status != 1 {
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.OldPassword)); err != nil {
		return nil, xerr.NewErrCode(xerr.OLD_PASSWORD_WRONG)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), 10)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	err = l.svcCtx.SysUserModel.UpdatePassword(l.ctx, user.Id, model.UserActor{UserId: in.GetActor().GetUserId(), OwnAccount: true}, user.PasswordHash, string(hash), func(ctx context.Context, userId int64) error {
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

func passwordMutationError(err error) error {
	if errors.Is(err, model.ErrDataScopeDenied) {
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if errors.Is(err, model.ErrUserDisabled) {
		return xerr.NewErrCode(xerr.USER_DISABLED)
	}
	if errors.Is(err, model.ErrPasswordChanged) {
		return xerr.NewErrCode(xerr.OLD_PASSWORD_WRONG)
	}
	return userMutationError(err)
}
