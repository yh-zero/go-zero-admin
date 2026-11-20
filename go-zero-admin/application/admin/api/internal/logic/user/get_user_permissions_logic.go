// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package user

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserPermissionsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserPermissionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserPermissionsLogic {
	return &GetUserPermissionsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserPermissionsLogic) GetUserPermissions(req *types.GetUserPermissionsReq) (resp *types.GetUserPermissionsResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	snapshot, err := l.svcCtx.UserRpc.GetUserPermissions(l.ctx, &adminpb.GetUserPermissionsRequest{
		Actor: middleware.ActorFromContext(l.ctx), UserId: req.Id, DeptId: req.DeptId,
	})
	if err != nil {
		return nil, err
	}
	out := &types.GetUserPermissionsResp{
		DeptId: snapshot.DeptId, Enabled: snapshot.Enabled,
		Depts: []*types.UserPermissionDept{}, Roles: []*types.UserPermissionRole{},
		Items: []*types.UserPermissionItem{}, DataScopes: append([]string{}, snapshot.DataScopes...),
		VisibleUsers: []*types.UserPermissionVisibleUser{},
	}
	if !snapshot.Enabled {
		return out, nil
	}
	for _, dept := range snapshot.Depts {
		out.Depts = append(out.Depts, &types.UserPermissionDept{Id: dept.Id, Name: dept.Name})
	}
	for _, role := range snapshot.Roles {
		out.Roles = append(out.Roles, &types.UserPermissionRole{Code: role.Code, Domain: role.Domain, DataScope: role.DataScope})
	}
	sub := fmt.Sprintf("user:%d", req.Id)
	domain := fmt.Sprintf("dept:%d", snapshot.DeptId)
	detailPath := fmt.Sprintf("/v1/sys/user/%d", req.Id)
	for _, operation := range []struct{ key, label, method, path, permCode string }{
		{"list", "用户列表", http.MethodGet, "/v1/sys/user/list", ""},
		{"detail", "用户详情", http.MethodGet, detailPath, ""},
		{"create", "新增用户", http.MethodPost, "/v1/sys/user/create", "sys:user:add"},
		{"update", "编辑用户", http.MethodPut, detailPath, "sys:user:edit"},
		{"delete", "删除用户", http.MethodDelete, detailPath, "sys:user:delete"},
		{"reset-password", "重置密码", http.MethodPost, detailPath + "/password", "sys:user:reset-password"},
	} {
		uiRoles := []string{}
		for _, resource := range snapshot.Resources {
			menu := operation.permCode == "" && resource.Type == 2 && strings.TrimRight(resource.Path, "/") == "/sys/user"
			button := operation.permCode != "" && resource.Type == 3 && resource.PermCode == operation.permCode
			if (menu || button) && !slices.Contains(uiRoles, resource.RoleCode) {
				uiRoles = append(uiRoles, resource.RoleCode)
			}
		}
		allowed, role, policyDomain, err := l.svcCtx.Enforcer.Explain(sub, domain, operation.path, operation.method)
		if err != nil {
			l.Error("could not evaluate user permissions")
			return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		if operation.key == "list" && allowed {
			out.VisibleUserCount = snapshot.VisibleUserCount
			for _, user := range snapshot.VisibleUsers {
				out.VisibleUsers = append(out.VisibleUsers, &types.UserPermissionVisibleUser{Id: user.Id, Username: user.Username})
			}
		}
		out.Items = append(out.Items, &types.UserPermissionItem{
			Key: operation.key, Label: operation.label, UiGranted: len(uiRoles) > 0, UiRoles: uiRoles,
			ApiAllowed: allowed, ApiRole: role, ApiDomain: policyDomain,
		})
	}
	return out, nil
}
