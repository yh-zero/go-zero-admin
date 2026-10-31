package appauthservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/app/rpc/internal/svc"
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
func (l *GetMyMembershipLogic) GetMyMembership(in *apppb.GetMyMembershipRequest) (*apppb.GetMyMembershipResponse, error) {
	u, _, err := CurrentAppUser(l.ctx, l.svcCtx, in.AccessToken)
	if err != nil {
		return nil, err
	}
	rows, _, err := l.svcCtx.AppMembershipGrantModel.List(l.ctx, int64(u.Id), "", true, 1, 0)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*apppb.MembershipGrantInfo, 0, len(rows))
	for _, row := range rows {
		list = append(list, MapMembership(row))
	}
	return &apppb.GetMyMembershipResponse{List: list}, nil
}
func MapMembership(r model.MembershipRow) *apppb.MembershipGrantInfo {
	return &apppb.MembershipGrantInfo{Id: r.Id, UserID: r.UserId, PlanCode: r.PlanCode, Benefits: r.Benefits, RequestId: r.RequestId, GrantedBy: r.GrantedBy.Int64, GrantedByAdmin: r.GrantedBy.Valid, StartsAt: r.StartsAt, ExpiresAt: r.ExpiresAt, Revoked: r.Revoked != 0, RevokedAt: r.RevokedAt, CreatedAt: r.CreatedAt}
}
