// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshAppTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 轮换 App 刷新令牌
func NewRefreshAppTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshAppTokenLogic {
	return &RefreshAppTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshAppTokenLogic) RefreshAppToken(req *types.AppRefreshRequest) (resp *types.AppAuthTokens, err error) {
	reply, err := l.svcCtx.AppAccountRPC.RefreshAppToken(l.ctx, &pb.AppRefreshRequest{RefreshToken: req.RefreshToken, IP: appbridge.Request(l.ctx).IP, UserAgent: appbridge.Request(l.ctx).UserAgent})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppAuthTokens](reply)
}
