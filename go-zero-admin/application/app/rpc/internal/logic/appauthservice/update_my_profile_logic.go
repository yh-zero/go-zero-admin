package appauthservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
	"unicode/utf8"
)

type UpdateMyProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateMyProfileLogic(ctx context.Context, s *svc.ServiceContext) *UpdateMyProfileLogic {
	return &UpdateMyProfileLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *UpdateMyProfileLogic) UpdateMyProfile(in *apppb.AppProfileRequest) (*apppb.AppProfileResponse, error) {
	u, _, err := CurrentAppUser(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	if in.Nickname != nil {
		if utf8.RuneCountInString(*in.Nickname) > 64 {
			return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
		}
		if err = l.svcCtx.AppUserModel.SetNickname(l.ctx, u.Id, *in.Nickname); err != nil {
			return nil, xerr.NewErrCode(xerr.DB_ERROR)
		}
		u.Nickname = *in.Nickname
	}
	info, err := profileInfo(l.ctx, l.svcCtx, u)
	if err != nil {
		return nil, err
	}
	return &apppb.AppProfileResponse{User: info}, nil
}
