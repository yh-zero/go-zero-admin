package profile

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateProfileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateProfileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateProfileLogic {
	return &UpdateProfileLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateProfileLogic) UpdateProfile(req *types.UpdateProfileReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor := middleware.ActorFromContext(l.ctx)
	_, err := l.svcCtx.ProfileRpc.UpdateProfile(l.ctx, &adminpb.UpdateProfileRequest{
		Actor: actor, Nickname: req.Nickname, RealName: req.RealName,
		Email: req.Email, Phone: req.Phone, Gender: req.Gender, AvatarFileId: req.AvatarFileId,
	})
	return &types.EmptyResp{}, err
}
