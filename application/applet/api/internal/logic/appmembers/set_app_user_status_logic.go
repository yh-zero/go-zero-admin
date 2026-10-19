// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appmembers

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SetAppUserStatusLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 审核或禁用恢复 App 账号
func NewSetAppUserStatusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SetAppUserStatusLogic {
	return &SetAppUserStatusLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SetAppUserStatusLogic) SetAppUserStatus(req *types.AppStatusRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppAccountRPC.SetAppUserStatus(l.ctx, &pb.AppStatusRequest{Actor: appbridge.Actor(l.ctx), UserID: req.UserID, Status: req.Status, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "操作成功"}, nil
}
