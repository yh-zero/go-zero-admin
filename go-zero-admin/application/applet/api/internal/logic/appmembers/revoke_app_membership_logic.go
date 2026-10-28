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

type RevokeAppMembershipLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 撤销指定会员授予
func NewRevokeAppMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeAppMembershipLogic {
	return &RevokeAppMembershipLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RevokeAppMembershipLogic) RevokeAppMembership(req *types.AppGrantRevokeRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppMembershipRPC.RevokeAppMembership(l.ctx, &pb.AppGrantRevokeRequest{Actor: appbridge.Actor(l.ctx), ID: req.ID, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "操作成功"}, nil
}
