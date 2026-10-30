package appadminservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type RevokeMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRevokeMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RevokeMembershipLogic {
	return &RevokeMembershipLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *RevokeMembershipLogic) RevokeMembership(in *apppb.RevokeMembershipRequest) (*apppb.NoDataResponse, error) {
	_, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"UPDATE app_membership_grant SET revoked=1, revoked_at=NOW() WHERE id=? AND revoked=0", in.GrantID)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
