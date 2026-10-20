// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appmembers

import (
	"context"

	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetAppUserAvatarLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 更新 App 用户头像，空地址恢复默认头像
func NewSetAppUserAvatarLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAppUserAvatarLogic {
	return &SetAppUserAvatarLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetAppUserAvatarLogic) SetAppUserAvatar(req *types.AppAvatarRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppAccountRPC.SetAppUserAvatar(l.ctx, &pb.AppAvatarRequest{Actor: appbridge.Actor(l.ctx), UserID: req.UserID, AvatarURL: req.AvatarURL})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "头像已更新"}, nil
}
