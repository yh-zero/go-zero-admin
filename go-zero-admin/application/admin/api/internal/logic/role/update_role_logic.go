package role

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleLogic {
	return &UpdateRoleLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateRoleLogic) UpdateRole(req *types.UpdateRoleReq) error {
	_, err := l.svcCtx.RoleRpc.UpdateRole(l.ctx, &adminpb.UpdateRoleRequest{
		RoleId: req.Id, Name: req.Name, Code: req.Code, Description: req.Description,
		DataScope: req.DataScope, Sort: req.Sort, Status: req.Status, MenuIds: req.MenuIds,
	})
	return err
}
