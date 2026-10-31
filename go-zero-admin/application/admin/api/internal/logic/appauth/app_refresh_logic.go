package appauth

import (
	"context"
	"fmt"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type AppRefreshLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppRefreshLogic(ctx context.Context, s *svc.ServiceContext) *AppRefreshLogic {
	return &AppRefreshLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppRefreshLogic) AppRefresh(req *types.AppRefreshReq) (*types.AppTokenResp, error) {
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	ip, ua := appRequestMeta(l.ctx)
	r, err := l.svcCtx.AppAuthRpc.AppRefresh(l.ctx, &apppb.AppRefreshRequest{RefreshToken: req.RefreshToken, IP: ip, UserAgent: ua})
	if err != nil {
		return nil, err
	}
	if r.User != nil {
		middleware.MarkAuditIdentity(l.ctx, 0, fmt.Sprintf("app:%d", r.User.Id), 0)
	}
	return &types.AppTokenResp{AccessToken: r.AccessToken, ExpiresAt: r.ExpiresAt, RefreshToken: r.RefreshToken, RefreshExpiresAt: r.RefreshExpiresAt, SessionId: r.SessionId, User: MapAppUser(r.User)}, nil
}
