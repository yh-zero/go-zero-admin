package userservicelogic

import (
	"context"
	"database/sql"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type CreateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
	return &CreateUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateUserLogic) CreateUser(in *adminpb.CreateUserRequest) (*adminpb.NoDataResponse, error) {
	_, err := l.svcCtx.SysUserModel.FindOneByUsername(l.ctx, in.Username)
	if err == nil {
		return nil, xerr.NewErrCode(xerr.USERNAME_EXISTS)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), 10)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}

	now := time.Now()
	user := &model.SysUser{
		Username:     in.Username,
		PasswordHash: string(hash),
		Nickname:     in.Nickname,
		Gender:       int64(in.Gender),
		Status:       int64(in.Status),
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	user.RealName = in.RealName
	user.Email = in.Email
	user.Phone = in.Phone
	if in.DeptId > 0 {
		user.DeptId = sql.NullInt64{Int64: in.DeptId, Valid: true}
	}
	if in.AvatarFileId > 0 {
		user.AvatarFileId = sql.NullInt64{Int64: in.AvatarFileId, Valid: true}
	}
	user.Remark = in.Remark

	_, err = l.svcCtx.SysUserModel.Insert(l.ctx, user)
	if err != nil {
		l.Errorf("insert user error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
