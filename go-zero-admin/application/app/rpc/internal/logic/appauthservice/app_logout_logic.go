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

func NewAppLogoutLogic(ctx context.Context, s *svc.ServiceContext) *AppLogoutLogic {
	return &AppLogoutLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppLogoutLogic) AppLogout(in *apppb.AppLogoutRequest) (*apppb.NoDataResponse, error) {
	u, session, err := CurrentAppUser(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	if in.AllDevices {
		err = l.svcCtx.Sessions.RevokeAll(l.ctx, int64(u.Id))
	} else {
		err = l.svcCtx.Sessions.Revoke(l.ctx, int64(u.Id), session.Id)
	}
	if err != nil {
		return nil, sessionError(err)
	}
	return &apppb.NoDataResponse{}, nil
}
