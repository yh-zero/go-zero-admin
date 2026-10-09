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

type PreviewPermissionRollbackLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewPermissionRollbackLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewPermissionRollbackLogic {
	return &PreviewPermissionRollbackLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *PreviewPermissionRollbackLogic) PreviewPermissionRollback(req *types.PreviewPermissionRollbackRequest) (resp *types.PreviewPermissionRollbackResponse, err error) {
	result, err := l.svcCtx.AppletPermissionRPC.PreviewPermissionRollback(l.ctx, &pb.PreviewPermissionRollbackRequest{ID: req.ID})
	if err != nil {
		return nil, err
	}
	return &types.PreviewPermissionRollbackResponse{Version: result.Version, Change: historyChange(result.Change), Allowed: result.Allowed, Reason: result.Reason}, nil
}
