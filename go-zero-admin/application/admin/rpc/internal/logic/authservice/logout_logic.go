package authservicelogic

import (
	"context"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"

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
	claims, err := jwtx.ParseAccessToken(l.svcCtx.Config.JwtAuth.AccessSecret, in.AccessToken)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if _, err := l.svcCtx.Sessions.Validate(l.ctx, claims.SessionId, claims.UserID, claims.ID); err != nil {
		return nil, sessionError(err)
	}
	if in.AllDevices {
		err = l.svcCtx.Sessions.RevokeAll(l.ctx, claims.UserID)
	} else {
		err = l.svcCtx.Sessions.Revoke(l.ctx, claims.UserID, claims.SessionId)
	}
	if err != nil {
		return nil, sessionError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}
