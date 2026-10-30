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

type ResetPasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewResetPasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ResetPasswordLogic {
	return &ResetPasswordLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ResetPasswordLogic) ResetPassword(in *adminpb.ResetPasswordRequest) (*adminpb.NoDataResponse, error) {
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.UserId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
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
