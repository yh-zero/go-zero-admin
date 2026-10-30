package appadminservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type ListAppUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListAppUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListAppUsersLogic {
	return &ListAppUsersLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListAppUsersLogic) ListAppUsers(in *apppb.AppAdminListRequest) (*apppb.AppAdminListResponse, error) {
	where := "deleted_at IS NULL"
	var args []interface{}
	if in.Status != "" {
		where += " AND status=?"
		args = append(args, in.Status)
	}
	if in.Account != "" {
		where += " AND account LIKE ?"
		args = append(args, "%"+in.Account+"%")
	}

	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM app_user WHERE "+where, args...)

	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	type userRow struct {
		Id           int64  `db:"id"`
		Account      string `db:"account"`
		IdentityKind string `db:"identity_kind"`
		Nickname     string `db:"nickname"`
		Status       string `db:"status"`
		ReviewReason string `db:"review_reason"`
		CreatedAt    int64  `db:"created_at"`
	}
	var rows []userRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, account, identity_kind, nickname, status, review_reason, UNIX_TIMESTAMP(created_at) FROM app_user WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	list := make([]*apppb.AppUserInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &apppb.AppUserInfo{
			Id: r.Id, Account: r.Account, IdentityKind: r.IdentityKind,
			Nickname: r.Nickname, Status: r.Status, ReviewReason: r.ReviewReason,
			CreatedAt: r.CreatedAt,
		})
	}
	return &apppb.AppAdminListResponse{List: list, Page: &apppb.PageResponse{Total: total}}, nil
}
