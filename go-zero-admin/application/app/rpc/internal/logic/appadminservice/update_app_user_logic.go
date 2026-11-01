package appadminservicelogic

import (
	"context"
	"unicode/utf8"

	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAppUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAppUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAppUserLogic {
	return &UpdateAppUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *UpdateAppUserLogic) UpdateAppUser(in *apppb.AppUpdateUserRequest) (*apppb.NoDataResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	if in.UserId <= 0 || in.Nickname != nil && utf8.RuneCountInString(*in.Nickname) > 64 || in.Remark != nil && utf8.RuneCountInString(*in.Remark) > 500 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if err := l.svcCtx.AppUserModel.PatchAdminProfile(l.ctx, uint64(in.UserId), in.Nickname, in.Remark); err != nil {
		return nil, appDataError(err)
	}
	return &apppb.NoDataResponse{}, nil
}
