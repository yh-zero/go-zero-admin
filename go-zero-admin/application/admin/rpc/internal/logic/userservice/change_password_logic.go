package userservicelogic

import (
	"context"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.Actor.UserId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.OldPassword)); err != nil {
		return nil, xerr.NewErrCode(xerr.OLD_PASSWORD_WRONG)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), 10)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	user.PasswordHash = string(hash)
	user.UpdatedAt = time.Now()
	err = l.svcCtx.SysUserModel.Update(l.ctx, user)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
