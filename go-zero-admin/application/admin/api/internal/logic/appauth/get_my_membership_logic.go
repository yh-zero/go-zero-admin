package appauth

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/pkg/xerr"
)

type GetMyMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyMembershipLogic(ctx context.Context, s *svc.ServiceContext) *GetMyMembershipLogic {
	return &GetMyMembershipLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *GetMyMembershipLogic) GetMyMembership() (*types.AppMyMembershipResp, error) {
	if l.svcCtx.AppAuthRpc == nil {
		return nil, xerr.NewErrCode(xerr.SERVICE_UNAVAILABLE)
	}

	r, err := l.svcCtx.AppAuthRpc.GetMyMembership(l.ctx, &apppb.GetMyMembershipRequest{AccessToken: appToken(l.ctx)})
	if err != nil {
		return nil, err
	}
	list := make([]types.MembershipGrantInfo, 0, len(r.List))
	for _, row := range r.List {
		list = append(list, MapMembership(row))
	}
	return &types.AppMyMembershipResp{List: list}, nil
}
