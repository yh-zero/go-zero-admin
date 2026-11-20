package userservicelogic

import (
	"context"
	"slices"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetUserPermissionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserPermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserPermissionsLogic {
	return &GetUserPermissionsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetUserPermissionsLogic) GetUserPermissions(in *adminpb.GetUserPermissionsRequest) (*adminpb.GetUserPermissionsResponse, error) {
	actor := model.UserActor{UserId: in.GetActor().GetUserId(), DeptId: in.GetActor().GetDeptId()}
	admin, err := l.svcCtx.SysUserModel.IsSuperAdmin(l.ctx, actor.UserId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if !admin {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	target, err := l.svcCtx.SysUserModel.FindScopedUser(l.ctx, in.GetUserId(), actor)
	if err != nil {
		return nil, passwordMutationError(err)
	}
	out := &adminpb.GetUserPermissionsResponse{Enabled: target.Status == 1}
	if !out.Enabled {
		return out, nil
	}
	targetAdmin, err := l.svcCtx.SysUserModel.IsSuperAdmin(l.ctx, in.GetUserId())
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	depts, err := l.svcCtx.SysDeptModel.ListMyDepartments(l.ctx, in.GetUserId(), targetAdmin)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	primary := target.DeptId.Int64
	if primary == 0 {
		out.Depts = append(out.Depts, &adminpb.UserPermissionDept{Name: "未分配部门"})
	}
	for _, dept := range depts {
		out.Depts = append(out.Depts, &adminpb.UserPermissionDept{Id: dept.Id, Name: dept.Name})
	}
	deptId, ok := permissionDepartment(primary, in.DeptId, out.Depts)
	if !ok {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	out.DeptId = deptId
	description, err := l.svcCtx.SysUserModel.DescribeUserScope(l.ctx, model.UserActor{UserId: in.GetUserId(), DeptId: deptId})
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	out.DataScopes, out.VisibleUserCount = description.Scopes, description.Count
	for _, user := range description.Users {
		out.VisibleUsers = append(out.VisibleUsers, &adminpb.UserPermissionVisibleUser{Id: user.Id, Username: user.Username})
	}
	codes := make([]string, 0, len(description.Roles))
	for _, role := range description.Roles {
		codes = append(codes, role.Code)
		out.Roles = append(out.Roles, &adminpb.UserPermissionRole{Code: role.Code, Domain: role.Domain, DataScope: strings.ToUpper(strings.TrimSpace(role.DataScope))})
	}
	resources, err := l.svcCtx.SysMenuModel.ListUserPermissionResources(l.ctx, codes)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	for _, resource := range resources {
		out.Resources = append(out.Resources, &adminpb.UserPermissionResource{
			RoleCode: resource.RoleCode, Path: resource.Path, PermCode: resource.PermCode, Type: resource.Type,
		})
	}
	return out, nil
}

func permissionDepartment(primary int64, selected *int64, depts []*adminpb.UserPermissionDept) (int64, bool) {
	available := func(id int64) bool {
		return slices.ContainsFunc(depts, func(dept *adminpb.UserPermissionDept) bool { return dept.Id == id })
	}
	if selected != nil {
		return *selected, available(*selected)
	}
	if available(primary) {
		return primary, true
	}
	if len(depts) > 0 {
		return depts[0].Id, true
	}
	// 和省略 X-Dept-Id 的请求保持一致。
	return 0, true
}
