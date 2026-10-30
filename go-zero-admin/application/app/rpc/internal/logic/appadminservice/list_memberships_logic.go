package appadminservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ListMembershipsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListMembershipsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListMembershipsLogic {
	return &ListMembershipsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

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

func (r *grantRow) toPb() *apppb.MembershipGrantInfo {
	return &apppb.MembershipGrantInfo{
		Id: r.Id, UserID: r.UserId, PlanCode: r.PlanCode, Benefits: r.Benefits,
		RequestId: r.RequestId, GrantedBy: r.GrantedBy,
		StartsAt: r.StartsAt, ExpiresAt: r.ExpiresAt,
		Revoked: r.Revoked == 1, CreatedAt: r.CreatedAt,
	}
}

func (l *ListMembershipsLogic) ListMemberships(in *apppb.ListMembershipRequest) (*apppb.ListMembershipResponse, error) {
	where := "1=1"
	var args []interface{}
	if in.UserID > 0 {
		where += " AND user_id=?"
		args = append(args, in.UserID)
	}
	if in.PlanCode != "" {
		where += " AND plan_code=?"
		args = append(args, in.PlanCode)
	}
	if in.ActiveOnly {
		where += " AND revoked=0 AND starts_at<=NOW() AND expires_at>=NOW()"
	}

	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM app_membership_grant WHERE "+where, args...)
	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var rows []grantRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, user_id, plan_code, benefits, request_id, COALESCE(granted_by,0), UNIX_TIMESTAMP(starts_at), UNIX_TIMESTAMP(expires_at), revoked, UNIX_TIMESTAMP(created_at) FROM app_membership_grant WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	list := make([]*apppb.MembershipGrantInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, r.toPb())
	}
	return &apppb.ListMembershipResponse{List: list, Page: &apppb.PageResponse{Total: total}}, nil
}
