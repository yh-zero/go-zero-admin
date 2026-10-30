package authservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type RefreshTokenLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RefreshTokenLogic) RefreshToken(in *adminpb.RefreshTokenRequest) (*adminpb.RefreshTokenResponse, error) {
	// 简版：refresh token 机制未完全实现，返回错误提示重新登录
	return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
}
