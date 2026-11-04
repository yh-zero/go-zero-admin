package appadmin

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type RevokeMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeMembershipLogic(ctx context.Context, s *svc.ServiceContext) *RevokeMembershipLogic {
	return &RevokeMembershipLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *RevokeMembershipLogic) RevokeMembership(req *types.AppRevokeMembershipReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.AppAdminRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	_, err := l.svcCtx.AppAdminRpc.RevokeMembership(l.ctx, &apppb.RevokeMembershipRequest{Actor: appAdminActor(l.ctx), GrantID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.EmptyResp{}, nil
}
