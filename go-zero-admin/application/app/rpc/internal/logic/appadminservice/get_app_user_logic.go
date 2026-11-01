package appadminservicelogic

import (
	"context"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAppUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppUserLogic {
	return &GetAppUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAppUserLogic) GetAppUser(in *apppb.AppAdminUserRequest) (*apppb.AppAdminUserResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	if in.UserId <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	row, err := l.svcCtx.AppUserModel.FindAdminProfile(l.ctx, uint64(in.UserId))
	if err != nil {
		return nil, appDataError(err)
	}
	info, err := adminUserInfo(l.ctx, l.svcCtx, row, nil)
	if err != nil {
		return nil, err
	}
	return &apppb.AppAdminUserResponse{User: info}, nil
}
