package appauth

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type UpdateMyProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateMyProfileLogic(ctx context.Context, s *svc.ServiceContext) *UpdateMyProfileLogic {
	return &UpdateMyProfileLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *UpdateMyProfileLogic) UpdateMyProfile(req *types.AppUpdateProfileReq) (*types.AppProfileResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	r, err := l.svcCtx.AppAuthRpc.UpdateMyProfile(l.ctx, &apppb.AppProfileRequest{AccessToken: appToken(l.ctx), Nickname: req.Nickname})
	if err != nil {
		return nil, err
	}
	return &types.AppProfileResp{User: MapAppUser(r.User)}, nil
}
