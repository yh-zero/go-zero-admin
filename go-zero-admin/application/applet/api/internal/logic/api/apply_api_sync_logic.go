package api

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type ApplyApiSyncLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewApplyApiSyncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ApplyApiSyncLogic {
	return &ApplyApiSyncLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *ApplyApiSyncLogic) ApplyApiSync(req *types.ApplyApiSyncRequest) (*types.ApplyApiSyncResponse, error) {
	data, err := l.svcCtx.AppletAPIRPC.ApplyApiSync(l.ctx, &pb.ApplyApiSyncRequest{Version: req.Version, Keys: req.Keys})
	if err != nil {
		return nil, err
	}
	return &types.ApplyApiSyncResponse{Added: data.Added, Updated: data.Updated}, nil
}
