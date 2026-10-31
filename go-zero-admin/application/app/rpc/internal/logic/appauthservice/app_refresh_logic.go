package appauthservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
	"time"
)

type AppRefreshLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppRefreshLogic(ctx context.Context, s *svc.ServiceContext) *AppRefreshLogic {
	return &AppRefreshLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppRefreshLogic) AppRefresh(in *apppb.AppRefreshRequest) (*apppb.AppRefreshResponse, error) {
	old, err := l.svcCtx.Sessions.GetByRefresh(l.ctx, in.RefreshToken)
	if err != nil {
		return nil, sessionError(err)
	}
	u, err := l.svcCtx.AppUserModel.FindOne(l.ctx, uint64(old.UserId))
	if errors.Is(err, model.ErrNotFound) {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if u.DeletedAt.Valid || u.Status != "active" {
		_ = l.svcCtx.Sessions.RevokeAll(l.ctx, old.UserId)
		return nil, xerr.NewErrCode(xerr.USER_DISABLED)
	}
	session, refresh, err := l.svcCtx.Sessions.Rotate(l.ctx, old, in.RefreshToken, in.IP, in.UserAgent)
	if err != nil {
		return nil, sessionError(err)
	}
	now := time.Now().Unix()
	expire := min(accessExpire(l.svcCtx), session.ExpiresAt-now)
	if expire <= 0 {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	token, err := jwtx.GenerateAppSessionToken(l.svcCtx.Config.JwtAuth.AccessSecret, session, u.Status, false, "", expire)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.TOKEN_GENERATE_ERROR)
	}
	return &apppb.AppRefreshResponse{AccessToken: token, ExpiresAt: now + expire, RefreshToken: refresh, RefreshExpiresAt: session.ExpiresAt, SessionId: session.Id, User: mapAppUser(u)}, nil
}
