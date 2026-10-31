// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package health

import (
	"context"
	"time"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type ReadyLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewReadyLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ReadyLogic {
	return &ReadyLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ReadyLogic) Ready() (resp *types.ReadyResp, err error) {
	ctx, cancel := context.WithTimeout(l.ctx, 2*time.Second)
	defer cancel()
	if err := l.svcCtx.Redis.Ping(ctx).Err(); err != nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	health, err := l.svcCtx.HealthRpc.CheckHealth(ctx, &adminpb.CheckHealthRequest{})
	if err != nil || !health.GetReady() {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	return &types.ReadyResp{Ready: true}, nil
}
