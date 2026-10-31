package appauth

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type GetMyProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyProfileLogic(ctx context.Context, s *svc.ServiceContext) *GetMyProfileLogic {
	return &GetMyProfileLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *GetMyProfileLogic) GetMyProfile() (*types.AppProfileResp, error) {
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	r, err := l.svcCtx.AppAuthRpc.GetMyProfile(l.ctx, &apppb.AppProfileRequest{AccessToken: appToken(l.ctx)})
	if err != nil {
		return nil, err
	}
	return &types.AppProfileResp{User: MapAppUser(r.User)}, nil
}
