package userservicelogic

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateUserLogic) UpdateUser(in *adminpb.UpdateUserRequest) (*adminpb.NoDataResponse, error) {
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.UserId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	user.Nickname = in.Nickname
	user.Gender = int64(in.Gender)
	user.Status = int64(in.Status)
	user.UpdatedAt = time.Now()
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
	err = l.svcCtx.SysUserModel.Update(l.ctx, user)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
