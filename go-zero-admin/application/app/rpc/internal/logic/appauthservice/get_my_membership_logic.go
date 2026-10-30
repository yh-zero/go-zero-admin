package appauthservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/apptoken"
)

type GetMyMembershipLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyMembershipLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyMembershipLogic {
	return &GetMyMembershipLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// GetMyMembership 查有效会员（Phase 2+3：快照优先，Redis 兜底）
func (l *GetMyMembershipLogic) GetMyMembership(in *apppb.GetMyMembershipRequest) (*apppb.GetMyMembershipResponse, error) {
	claims, err := apptoken.Claims(l.svcCtx.Config.JwtAuth.AccessSecret, in.AccessToken)
	if err != nil {
		return nil, err
	}
	if err := apptoken.MustActive(claims); err != nil {
		return nil, err
	}

	// Phase 2: 快照判断（零查库，绝大多数情况走这里）
	if !apptoken.IsVip(l.ctx, l.svcCtx.Redis, claims) {
		return &apppb.GetMyMembershipResponse{List: []*apppb.MembershipGrantInfo{}}, nil
	}

	// 是会员：查明细（带 row struct，遵守规则 8）
	type grantRow struct {
		Id        int64  `db:"id"`
		UserId    int64  `db:"user_id"`
		PlanCode  string `db:"plan_code"`
		Benefits  string `db:"benefits"`
		RequestId string `db:"request_id"`
		GrantedBy int64  `db:"granted_by"`
		StartsAt  int64  `db:"starts_at"`
		ExpiresAt int64  `db:"expires_at"`
		Revoked   int64  `db:"revoked"`
		CreatedAt int64  `db:"created_at"`
	}
	var rows []grantRow
	err = l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, user_id, plan_code, benefits, request_id, COALESCE(granted_by,0), UNIX_TIMESTAMP(starts_at), UNIX_TIMESTAMP(expires_at), revoked, UNIX_TIMESTAMP(created_at) FROM app_membership_grant WHERE user_id=? AND revoked=0 AND starts_at<=NOW() AND expires_at>=NOW()",
		claims.UserID)
	if err != nil {
		return &apppb.GetMyMembershipResponse{List: []*apppb.MembershipGrantInfo{}}, nil
	}

	grants := make([]*apppb.MembershipGrantInfo, 0, len(rows))
	for _, r := range rows {
		grants = append(grants, &apppb.MembershipGrantInfo{
			Id: r.Id, UserID: r.UserId, PlanCode: r.PlanCode, Benefits: r.Benefits,
			RequestId: r.RequestId, GrantedBy: r.GrantedBy,
			StartsAt: r.StartsAt, ExpiresAt: r.ExpiresAt,
			Revoked: r.Revoked == 1, CreatedAt: r.CreatedAt,
		})
	}
	return &apppb.GetMyMembershipResponse{List: grants}, nil
}
