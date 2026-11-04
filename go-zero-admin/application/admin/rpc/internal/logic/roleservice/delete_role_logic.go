package roleservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRoleLogic {
	return &DeleteRoleLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteRoleLogic) DeleteRole(in *adminpb.DeleteRoleRequest) (*adminpb.NoDataResponse, error) {
	if err := requireRoleActor(l.ctx, l.svcCtx, in.GetActor()); err != nil {
		return nil, err
	}
	err := l.svcCtx.SysRoleModel.SoftDeleteWithRelations(l.ctx, uint64(in.RoleId))
	if model.MutationCommitted(err) {
		l.svcCtx.PublishPolicyChanged(l.ctx)
	}
	if err != nil {
		return nil, roleMutationError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}

func roleMutationError(err error) error {
	if errors.Is(err, sqlx.ErrNotFound) {
		return xerr.NewErrCode(xerr.ROLE_NOT_FOUND)
	}
	if errors.Is(err, model.ErrInvalidMutation) {
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, err.Error())
	}
	return xerr.NewErrCode(xerr.DB_ERROR)
}
