package appadminservicelogic

import (
	"context"
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
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
func (l *RevokeMembershipLogic) RevokeMembership(in *apppb.RevokeMembershipRequest) (*apppb.NoDataResponse, error) {
	if err := requireActor(in.Actor); err != nil {
		return nil, err
	}
	if in.GrantID <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if err := l.svcCtx.AppMembershipGrantModel.Revoke(l.ctx, uint64(in.GrantID)); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.NOT_FOUND)
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &apppb.NoDataResponse{}, nil
}
