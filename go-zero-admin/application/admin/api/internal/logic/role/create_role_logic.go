package role

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateRoleLogic {
	return &CreateRoleLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateRoleLogic) CreateRole(req *types.CreateRoleReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	_, err := l.svcCtx.RoleRpc.CreateRole(l.ctx, &adminpb.CreateRoleRequest{
		Actor: middleware.ActorFromContext(l.ctx),
		Name:  req.Name, Code: req.Code, Description: req.Description,
		DataScope: req.DataScope, Sort: req.Sort, Status: req.Status, MenuIds: req.MenuIds,
	})
	return &types.EmptyResp{}, err
}
