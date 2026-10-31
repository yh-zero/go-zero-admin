package tenantservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/tenant/rpc/internal/model"
	"go-zero-admin/application/tenant/rpc/internal/svc"
	"go-zero-admin/application/tenant/rpc/tenantpb"
	"go-zero-admin/pkg/xerr"
)

type GetTenantListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetTenantListLogic(ctx context.Context, s *svc.ServiceContext) *GetTenantListLogic {
	return &GetTenantListLogic{ctx: ctx, svcCtx: s, Logger: logx.WithContext(ctx)}
}
func (l *GetTenantListLogic) GetTenantList(in *tenantpb.GetTenantListRequest) (*tenantpb.GetTenantListResponse, error) {
	if in.Actor == nil || in.Actor.UserID <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	page, size := in.GetPage().GetPage(), in.GetPage().GetPageSize()
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	rows, total, err := l.svcCtx.TenantRegistryModel.List(l.ctx, page, size, in.Status)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*tenantpb.TenantInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, mapTenantRow(r))
	}
	return &tenantpb.GetTenantListResponse{List: list, Page: &tenantpb.PageResponse{Total: total}}, nil
}
func mapTenantRow(r model.TenantListRow) *tenantpb.TenantInfo {
	return &tenantpb.TenantInfo{Id: r.Id, Slug: r.Slug, Domain: r.Domain, ComposeProject: r.ComposeProject, Status: r.Status, ErrorMsg: r.ErrorMsg, CreatedBy: r.CreatedBy, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, AdminAccountReady: r.AdminAccountReady == 1}
}
func mapTenant(r *model.TenantRegistry) *tenantpb.TenantInfo {
	return &tenantpb.TenantInfo{Id: int64(r.Id), Slug: r.Slug, Domain: r.Domain, ComposeProject: r.ComposeProject, Status: r.Status, ErrorMsg: r.ErrorMsg, CreatedBy: r.CreatedBy.Int64, CreatedAt: r.CreatedAt.Unix(), UpdatedAt: r.UpdatedAt.Unix(), AdminAccountReady: r.AdminAccountReady == 1}
}
