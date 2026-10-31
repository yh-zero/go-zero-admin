// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package api

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/rpc/adminpb"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SyncApiLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSyncApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncApiLogic {
	return &SyncApiLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SyncApiLogic) SyncApi(req *types.SyncApiReq) (resp *types.SyncApiResp, err error) {
	result, err := l.svcCtx.ApiRpc.SyncApi(l.ctx, &adminpb.SyncApiRequest{Actor: middleware.ActorFromContext(l.ctx), Routes: l.svcCtx.ApiRoutes})
	if err != nil {
		return nil, err
	}
	return &types.SyncApiResp{Added: result.Added, Updated: result.Updated, Removed: result.Removed}, nil
}
