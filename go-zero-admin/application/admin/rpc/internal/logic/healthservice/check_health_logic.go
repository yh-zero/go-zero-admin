package healthservicelogic

import (
	"context"
	"time"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type CheckHealthLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCheckHealthLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CheckHealthLogic {
	return &CheckHealthLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *CheckHealthLogic) CheckHealth(in *adminpb.CheckHealthRequest) (*adminpb.CheckHealthResponse, error) {
	ctx, cancel := context.WithTimeout(l.ctx, 2*time.Second)
	defer cancel()
	var row struct {
		Value int64 `db:"value"`
	}
	if err := l.svcCtx.DB.QueryRowCtx(ctx, &row, "SELECT 1 AS value"); err != nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	if !l.svcCtx.Redis.PingCtx(ctx) {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}
	return &adminpb.CheckHealthResponse{Ready: true}, nil
}
