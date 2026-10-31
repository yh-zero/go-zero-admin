// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package auth

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/jwtx"

	"github.com/zeromicro/go-zero/core/logx"
)

type RefreshTokenLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRefreshTokenLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RefreshTokenLogic {
	return &RefreshTokenLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RefreshTokenLogic) RefreshToken(req *types.RefreshTokenReq) (resp *types.LoginResp, err error) {
	ip, _ := l.ctx.Value(middleware.CtxKeyClientIP).(string)
	ua, _ := l.ctx.Value(middleware.CtxKeyUserAgent).(string)
	rpcResp, err := l.svcCtx.AuthRpc.RefreshToken(l.ctx, &adminpb.RefreshTokenRequest{RefreshToken: req.RefreshToken, Ip: ip, UserAgent: ua})
	if err != nil {
		return nil, err
	}
	if claims, parseErr := jwtx.ParseAccessToken(l.svcCtx.Config.Auth.AccessSecret, rpcResp.AccessToken); parseErr == nil {
		middleware.MarkAuditIdentity(l.ctx, claims.UserID, claims.Username, claims.DeptId)
	}
	return &types.LoginResp{AccessToken: rpcResp.AccessToken, ExpiresAt: rpcResp.ExpiresAt, RefreshToken: rpcResp.RefreshToken, RefreshExpiresAt: rpcResp.RefreshExpiresAt, SessionId: rpcResp.SessionId}, nil
}
