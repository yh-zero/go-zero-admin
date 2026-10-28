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

type GrantAppMembershipLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 人工开通或续期会员
func NewGrantAppMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GrantAppMembershipLogic {
	return &GrantAppMembershipLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GrantAppMembershipLogic) GrantAppMembership(req *types.AppGrantRequest) (resp *types.AppGrant, err error) {
	reply, err := l.svcCtx.AppMembershipRPC.GrantAppMembership(l.ctx, &pb.AppGrantRequest{Actor: appbridge.Actor(l.ctx), UserID: req.UserID, PlanID: req.PlanID, Days: req.Days, RequestID: req.RequestID, Reason: req.Reason})
	if err != nil {
		return nil, err
	}
	return appbridge.Copy[types.AppGrant](reply)
}
