package userservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteUserLogic) DeleteUser(in *adminpb.DeleteUserRequest) (*adminpb.NoDataResponse, error) {
	_, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(in.UserId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
	}
	// 软删除
	now := time.Now()
	_, err = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_user SET deleted_at=? WHERE id=?", now, in.UserId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
