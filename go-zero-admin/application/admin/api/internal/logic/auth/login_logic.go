package auth

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/jwtx"
)

type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *LoginLogic) Login(req *types.LoginReq) (resp *types.LoginResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	ip, _ := l.ctx.Value(middleware.CtxKeyClientIP).(string)
	ua, _ := l.ctx.Value(middleware.CtxKeyUserAgent).(string)
	rpcResp, err := l.svcCtx.AuthRpc.Login(l.ctx, &adminpb.LoginRequest{
		Username:  req.Username,
		Password:  req.Password,
		CaptchaId: req.CaptchaID,
		Captcha:   req.Captcha,
		Ip:        ip,
		UserAgent: ua,
	})
	if err != nil {
		return nil, err
	}
	if claims, parseErr := jwtx.ParseAccessToken(l.svcCtx.Config.Auth.AccessSecret, rpcResp.AccessToken); parseErr == nil {
		middleware.MarkAuditIdentity(l.ctx, claims.UserID, claims.Username, claims.DeptId)
	}
	return &types.LoginResp{
		AccessToken:      rpcResp.AccessToken,
		ExpiresAt:        rpcResp.ExpiresAt,
		RefreshToken:     rpcResp.RefreshToken,
		RefreshExpiresAt: rpcResp.RefreshExpiresAt,
		SessionId:        rpcResp.SessionId,
	}, nil
}
