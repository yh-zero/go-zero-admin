package tenantservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"

	"go-zero-admin/pkg/xerr"
)

type GetTenantListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTenantListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetTenantListLogic {
	return &GetTenantListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetTenantListLogic) GetTenantList(in *tenantpb.GetTenantListRequest) (*tenantpb.GetTenantListResponse, error) {
	where := "deleted_at IS NULL"
	var args []interface{}
	if in.Status != "" {
		where += " AND status=?"
		args = append(args, in.Status)
	}
	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM tenant_registry WHERE "+where, args...)
	page := in.GetPage().GetPage()
	pageSize := in.GetPage().GetPageSize()
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	var rows []tenantRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, slug, domain, compose_project, status, error_msg, created_by, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at), COALESCE(admin_account_ready,0) FROM tenant_registry WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*tenantpb.TenantInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, mapTenant(&r))
	}
	return &tenantpb.GetTenantListResponse{List: list, Page: &tenantpb.PageResponse{Total: total}}, nil
}

type tenantRow struct {
	Id                int64  `db:"id"`
	Slug              string `db:"slug"`
	Domain            string `db:"domain"`
	ComposeProject    string `db:"compose_project"`
	Status            string `db:"status"`
	ErrorMsg          string `db:"error_msg"`
	CreatedBy         int64  `db:"created_by"`
	CreatedAt         int64  `db:"created_at"`
	UpdatedAt         int64  `db:"updated_at"`
	AdminAccountReady int64  `db:"admin_account_ready"`
}

func mapTenant(r *tenantRow) *tenantpb.TenantInfo {
	return &tenantpb.TenantInfo{
		Id: r.Id, Slug: r.Slug, Domain: r.Domain, ComposeProject: r.ComposeProject,
		Status: r.Status, ErrorMsg: r.ErrorMsg, CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, AdminAccountReady: r.AdminAccountReady == 1,
	}
}
