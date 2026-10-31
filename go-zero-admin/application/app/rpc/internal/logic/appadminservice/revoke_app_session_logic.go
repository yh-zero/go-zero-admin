package appadminservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/jwtx"
	"go-zero-admin/pkg/xerr"
)

type RevokeAppSessionLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeAppSessionLogic(ctx context.Context, s *svc.ServiceContext) *RevokeAppSessionLogic {
	return &RevokeAppSessionLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *RevokeAppSessionLogic) RevokeAppSession(in *apppb.AppAdminRevokeRequest) (*apppb.NoDataResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	if _, err := liveAppUser(l.ctx, l.svcCtx, in.UserId); err != nil {
		return nil, err
	}
	if in.SessionID == "" || len(in.SessionID) > 128 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if err := l.svcCtx.Sessions.Revoke(l.ctx, in.UserId, in.SessionID); err != nil {
		if errors.Is(err, jwtx.ErrSessionInvalid) {
			return nil, xerr.NewErrCode(xerr.NOT_FOUND)
		}
		return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
