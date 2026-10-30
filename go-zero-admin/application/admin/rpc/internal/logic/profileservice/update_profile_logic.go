package profileservicelogic

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProfileLogic {
	return &UpdateProfileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateProfileLogic) UpdateProfile(in *adminpb.UpdateProfileRequest) (*adminpb.NoDataResponse, error) {
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.Actor.UserId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	user.Nickname = in.Nickname
	user.Gender = int64(in.Gender)
	user.UpdatedAt = time.Now()
	user.RealName = in.RealName
	user.Email = in.Email
	user.Phone = in.Phone
	if in.AvatarFileId > 0 {
		user.AvatarFileId = sql.NullInt64{Int64: in.AvatarFileId, Valid: true}
	}
	err = l.svcCtx.SysUserModel.Update(l.ctx, user)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
