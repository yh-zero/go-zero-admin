package authservicelogic

import (
	"context"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/jwtx"
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
	session, err := l.svcCtx.Sessions.GetByRefresh(l.ctx, in.RefreshToken)
	if err != nil {
		return nil, sessionError(err)
	}
	user, err := l.svcCtx.SysUserModel.FindOne(l.ctx, uint64(session.UserId))
	if err != nil {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if user.DeletedAt.Valid || user.Status != 1 {
		_ = l.svcCtx.Sessions.RevokeAll(l.ctx, session.UserId)
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}
	session.Username, session.DeptId = user.Username, user.DeptId.Int64
	session, refresh, err := l.svcCtx.Sessions.Rotate(l.ctx, session, in.RefreshToken, in.Ip, in.UserAgent)
	if err != nil {
		return nil, sessionError(err)
	}
	expire := min(l.svcCtx.Config.JwtAuth.AccessExpire, session.ExpiresAt-time.Now().Unix())
	token, err := jwtx.GenerateSessionAccessToken(l.svcCtx.Config.JwtAuth.AccessSecret, session, expire)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.TOKEN_GENERATE_ERROR)
	}
	return &adminpb.RefreshTokenResponse{AccessToken: token, ExpiresAt: time.Now().Unix() + expire, RefreshToken: refresh, RefreshExpiresAt: session.ExpiresAt, SessionId: session.Id}, nil
}
