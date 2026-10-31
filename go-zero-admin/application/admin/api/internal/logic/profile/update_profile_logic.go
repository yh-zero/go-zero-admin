package profile

import (
	"context"

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

func (l *UpdateProfileLogic) UpdateProfile(req *types.UpdateProfileReq) error {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	actor := &adminpb.Actor{UserId: userID}
	_, err := l.svcCtx.ProfileRpc.UpdateProfile(l.ctx, &adminpb.UpdateProfileRequest{
		Actor: actor, Nickname: req.Nickname, RealName: req.RealName,
		Email: req.Email, Phone: req.Phone, AvatarFileId: 0, // TODO: 前端传 file_id 后映射 Gender: req.Gender,
	})
	return err
}
