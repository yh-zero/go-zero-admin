package appauth

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type AppLogoutLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAppLogoutLogic(ctx context.Context, s *svc.ServiceContext) *AppLogoutLogic {
	return &AppLogoutLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *AppLogoutLogic) AppLogout(req *types.AppLogoutReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	_, err := l.svcCtx.AppAuthRpc.AppLogout(l.ctx, &apppb.AppLogoutRequest{AccessToken: appToken(l.ctx), AllDevices: req.AllDevices})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
