package appadmin

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type GrantMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGrantMembershipLogic(ctx context.Context, s *svc.ServiceContext) *GrantMembershipLogic {
	return &GrantMembershipLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *GrantMembershipLogic) GrantMembership(req *types.AppGrantMembershipReq) (*types.EmptyResp, error) {
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	_, err := l.svcCtx.AppAdminRpc.GrantMembership(l.ctx, &apppb.GrantMembershipRequest{Actor: appAdminActor(l.ctx), UserID: req.UserId, PlanCode: req.PlanCode, Benefits: req.Benefits, RequestID: req.RequestId, DurationDays: req.DurationDays})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
