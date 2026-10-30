package role

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateRoleMenusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateRoleMenusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateRoleMenusLogic {
	return &UpdateRoleMenusLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateRoleMenusLogic) UpdateRoleMenus(req *types.UpdateRoleMenusReq) error {
	_, err := l.svcCtx.RoleRpc.UpdateRoleMenus(l.ctx, &adminpb.UpdateRoleMenusRequest{
		RoleId: req.Id, MenuIds: req.MenuIds,
	})
	return err
}
