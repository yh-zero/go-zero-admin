package appadminservicelogic

import (
	"context"
	"database/sql"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type SetAppUserAvatarLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSetAppUserAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAppUserAvatarLogic {
	return &SetAppUserAvatarLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SetAppUserAvatarLogic) SetAppUserAvatar(in *apppb.AppSetAvatarRequest) (*apppb.NoDataResponse, error) {
	user, err := l.svcCtx.AppUserModel.FindOne(l.ctx, uint64(in.UserID))
	if err != nil {
		return nil, xerr.NewErrCodeMsg(400, "用户不存在")
	}
	if in.FileID > 0 {
		user.AvatarFileId = sql.NullInt64{Int64: in.FileID, Valid: true}
	} else {
		user.AvatarFileId = sql.NullInt64{}
	}
	user.UpdatedAt = time.Now()
	err = l.svcCtx.AppUserModel.Update(l.ctx, user)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
