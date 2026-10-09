// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package permission

import (
	"context"

	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyPermissionRollbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApplyPermissionRollbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyPermissionRollbackLogic {
	return &ApplyPermissionRollbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ApplyPermissionRollbackLogic) ApplyPermissionRollback(req *types.ApplyPermissionRollbackRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletPermissionRPC.ApplyPermissionRollback(l.ctx, &pb.ApplyPermissionRollbackRequest{ID: req.ID, ExpectedRevision: req.ExpectedRevision, Version: req.Version})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "回滚成功"}, nil
}
