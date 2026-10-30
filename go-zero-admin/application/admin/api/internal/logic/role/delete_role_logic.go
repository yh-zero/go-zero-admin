package role

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRoleLogic {
	return &DeleteRoleLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteRoleLogic) DeleteRole(req *types.DeleteRoleReq) error {
	_, err := l.svcCtx.RoleRpc.DeleteRole(l.ctx, &adminpb.DeleteRoleRequest{RoleId: req.Id})
	return err
}
