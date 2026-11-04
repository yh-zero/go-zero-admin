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

type DeleteRoleLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteRoleLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteRoleLogic {
	return &DeleteRoleLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteRoleLogic) DeleteRole(req *types.DeleteRoleReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	_, err := l.svcCtx.RoleRpc.DeleteRole(l.ctx, &adminpb.DeleteRoleRequest{
		Actor: middleware.ActorFromContext(l.ctx), RoleId: req.Id})
	return &types.EmptyResp{}, err
}
