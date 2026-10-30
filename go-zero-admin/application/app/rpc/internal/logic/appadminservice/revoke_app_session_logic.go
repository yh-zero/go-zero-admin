package appadminservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
)

type RevokeAppSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeAppSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeAppSessionLogic {
	return &RevokeAppSessionLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RevokeAppSessionLogic) RevokeAppSession(in *apppb.AppAdminRevokeRequest) (*apppb.NoDataResponse, error) {
	// 简版：JWT 无状态，会话撤销需 Redis 黑名单（同 admin 模块，先占位）
	return &apppb.NoDataResponse{}, nil
}
