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
	// Gender 必须转发：RPC 侧全字段覆盖，丢字段会把性别清零（历史缺陷 R08）
	// AvatarUrl→AvatarFileId 的映射待 C02 明确（头像编辑未开放，传 0=不更新）
	_, err := l.svcCtx.ProfileRpc.UpdateProfile(l.ctx, &adminpb.UpdateProfileRequest{
		Actor: actor, Nickname: req.Nickname, RealName: req.RealName,
		Email: req.Email, Phone: req.Phone, Gender: req.Gender, AvatarFileId: 0,
	})
	return err
}
