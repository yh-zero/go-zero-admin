package authservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type RevokeDeviceSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeDeviceSessionLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeDeviceSessionLogic {
	return &RevokeDeviceSessionLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RevokeDeviceSessionLogic) RevokeDeviceSession(in *adminpb.RevokeDeviceSessionRequest) (*adminpb.NoDataResponse, error) {
	if in.Actor == nil || in.Actor.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	if err := l.svcCtx.Sessions.Revoke(l.ctx, in.Actor.UserId, in.SessionId); err != nil {
		return nil, sessionError(err)
	}
	return &adminpb.NoDataResponse{}, nil
}
