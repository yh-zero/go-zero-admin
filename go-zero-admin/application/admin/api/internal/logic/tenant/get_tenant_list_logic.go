package tenant

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
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
func (l *GetTenantListLogic) GetTenantList(req *types.GetTenantListReq) (*types.GetTenantListResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := authorizeTenant(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	result, err := l.svcCtx.TenantRpc.GetTenantList(l.ctx, &tenantpb.GetTenantListRequest{Actor: actor, Page: &tenantpb.PageRequest{Page: req.Page, PageSize: req.PageSize}, Status: req.Status})
	if err != nil {
		return nil, err
	}
	return &types.GetTenantListResp{List: mapTenantList(result.List), Total: result.GetPage().GetTotal()}, nil
}

func authorizeTenant(ctx context.Context, s *svc.ServiceContext) (*tenantpb.Actor, error) {
	actor := middleware.ActorFromContext(ctx)
	current, err := s.UserRpc.GetUser(ctx, &adminpb.GetUserRequest{Actor: actor, UserId: actor.UserId})
	if err != nil {
		return nil, err
	}
	if current.User == nil || !current.User.IsSuperAdmin {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if s.TenantRpc == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "当前实例未启用租户管理")
	}
	return &tenantpb.Actor{UserID: actor.UserId, Username: actor.Username, DeptId: actor.DeptId, IP: actor.Ip, UserAgent: actor.UserAgent}, nil
}
func mapTenantInfo(t *tenantpb.TenantInfo) *types.TenantInfo {
	if t == nil {
		return nil
	}
	return &types.TenantInfo{Id: t.Id, Slug: t.Slug, Domain: t.Domain, ComposeProject: t.ComposeProject, Status: t.Status, ErrorMsg: t.ErrorMsg, CreatedBy: t.CreatedBy, CreatedAt: t.CreatedAt, UpdatedAt: t.UpdatedAt, AdminAccountReady: t.AdminAccountReady}
}
func mapTenantList(list []*tenantpb.TenantInfo) []*types.TenantInfo {
	result := make([]*types.TenantInfo, 0, len(list))
	for _, item := range list {
		result = append(result, mapTenantInfo(item))
	}
	return result
}
