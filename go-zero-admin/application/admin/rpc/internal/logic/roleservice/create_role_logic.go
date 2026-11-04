package roleservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-zero-admin/application/admin/rpc/internal/model"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type CreateRoleLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRoleLogic {
	return &CreateRoleLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateRoleLogic) CreateRole(in *adminpb.CreateRoleRequest) (*adminpb.NoDataResponse, error) {
	if err := requireRoleActor(l.ctx, l.svcCtx, in.GetActor()); err != nil {
		return nil, err
	}
	status, err := model.CreateStatus(in.Status)
	if err != nil {
		return nil, roleMutationError(err)
	}
	if in.DataScope == "" {
		in.DataScope = "SELF"
	}
	_, err = l.svcCtx.SysRoleModel.FindOneByCode(l.ctx, in.Code)
	if err == nil {
		return nil, xerr.NewErrCode(xerr.ROLE_CODE_EXISTS)
	}
	if !errors.Is(err, sqlx.ErrNotFound) {
		return nil, roleMutationError(err)
	}
	err = l.svcCtx.SysRoleModel.CreateWithMenus(l.ctx, &model.SysRole{
		Name: in.Name, Code: in.Code, Description: in.Description,
		DataScope: in.DataScope, Sort: int64(in.Sort), Status: status,
	}, in.MenuIds)
	if model.MutationCommitted(err) {
		l.svcCtx.PublishPolicyChanged(l.ctx)
	}
	if err != nil {
		return nil, roleMutationError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}
