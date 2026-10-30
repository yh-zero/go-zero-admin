package authservicelogic

import (
	"context"
	"go-zero-admin/pkg/jwtx"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type LogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLogoutLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LogoutLogic {
	return &LogoutLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *LogoutLogic) Logout(in *adminpb.LogoutRequest) (*adminpb.NoDataResponse, error) {
	// 解析 token 拿 jti 加黑名单
	claims, err := jwtx.ParseAccessToken(l.svcCtx.Config.JwtAuth.AccessSecret, in.AccessToken)
	if err != nil {
		return &adminpb.NoDataResponse{}, nil
	} // token 无效也允许登出
	// 简版：直接返回（完整实现需 Redis 黑名单）
	_ = claims
	return &adminpb.NoDataResponse{}, nil
}
