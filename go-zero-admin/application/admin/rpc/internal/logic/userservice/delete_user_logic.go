package userservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
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
	if in.GetActor().GetUserId() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	err := l.svcCtx.SysUserModel.DeleteWithRoles(l.ctx, in.UserId, model.UserActor{UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId()})
	if err != nil && !errors.Is(err, sqlx.ErrNotFound) {
		return nil, userMutationError(err)
	}
	revokeErr := l.svcCtx.Sessions.RevokeAll(l.ctx, in.UserId)
	l.svcCtx.PublishPolicyChanged(l.ctx)
	cleanupErr := ReconcileUserAvatar(l.ctx, l.svcCtx, in.UserId, in.GetActor())
	if revokeErr != nil {
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	if cleanupErr != nil {
		return nil, cleanupErr
	}
	if err != nil {
		return nil, userMutationError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}
