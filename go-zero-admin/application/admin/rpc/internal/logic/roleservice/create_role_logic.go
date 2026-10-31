package roleservicelogic

import (
	"context"

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
	_, err := l.svcCtx.SysRoleModel.FindOneByCode(l.ctx, in.Code)
	if err == nil {
		return nil, xerr.NewErrCode(xerr.ROLE_CODE_EXISTS)
	}
	err = l.svcCtx.SysRoleModel.CreateWithMenus(l.ctx, &model.SysRole{
		Name: in.Name, Code: in.Code, Description: in.Description,
		DataScope: in.DataScope, Sort: int64(in.Sort), Status: int64(in.Status),
	}, in.MenuIds)
	if err != nil {
		return nil, roleMutationError(err)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
