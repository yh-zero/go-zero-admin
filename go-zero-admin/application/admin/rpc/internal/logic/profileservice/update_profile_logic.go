package profileservicelogic

import (
	"context"
	"errors"
	userservicelogic "go-zero-admin/application/admin/rpc/internal/logic/userservice"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UpdateProfileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProfileLogic {
	return &UpdateProfileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateProfileLogic) UpdateProfile(in *adminpb.UpdateProfileRequest) (*adminpb.NoDataResponse, error) {
	userId := in.GetActor().GetUserId()
	err := userservicelogic.UpdateUserWithAvatar(l.ctx, l.svcCtx, userId, in.GetActor(), model.UserPatch{
		OwnAccount: true,
		Nickname:   in.Nickname, RealName: in.RealName, Email: in.Email, Phone: in.Phone, Gender: in.Gender, AvatarFileId: in.AvatarFileId,
	})
	if err != nil {
		if business, ok := xerr.AsCodeError(err); ok {
			return nil, business
		}
		if errors.Is(err, model.ErrInvalidMutation) {
			return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, err.Error())
		}
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.USER_NOT_FOUND)
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
