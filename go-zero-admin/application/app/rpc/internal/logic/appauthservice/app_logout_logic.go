package appauthservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
)

type AppLogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AppLogoutLogic {
	return &AppLogoutLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *AppLogoutLogic) AppLogout(in *apppb.AppLogoutRequest) (*apppb.NoDataResponse, error) {
	// 简版：JWT 无状态，客户端丢弃 token 即可；完整版可加 Redis 黑名单
	return &apppb.NoDataResponse{}, nil
}
