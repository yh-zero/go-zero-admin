package user

import (
	"context"
	"testing"

	"github.com/go-playground/validator/v10"
	"google.golang.org/grpc"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/client/userservice"
	"go-zero-admin/pkg/casbinauth"
)

type permissionsUserRPC struct {
	userservice.UserService
}

func (permissionsUserRPC) GetUserPermissions(_ context.Context, req *adminpb.GetUserPermissionsRequest, _ ...grpc.CallOption) (*adminpb.GetUserPermissionsResponse, error) {
	return &adminpb.GetUserPermissionsResponse{
		Enabled: true, DeptId: req.GetDeptId(), DataScopes: []string{"SELF"},
		Resources:        []*adminpb.UserPermissionResource{{RoleCode: "reader", Path: "/sys/user", Type: 2}},
		VisibleUserCount: 1,
		VisibleUsers:     []*adminpb.UserPermissionVisibleUser{{Id: req.UserId, Username: "demo"}},
	}, nil
}

func TestUserPermissionSeparatesMenuFromAPIAccess(t *testing.T) {
	enforcer, err := casbinauth.NewHolder([]casbinauth.PolicyRule{
		{Ptype: "g", V0: "user:7", V1: "reader", V2: "dept:3"},
		{Ptype: "p", V0: "reader", V1: "*", V2: "/v1/sys/user/list", V3: "GET"},
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := &svc.ServiceContext{UserRpc: permissionsUserRPC{}, Validator: validator.New(), Enforcer: enforcer}
	logic := NewGetUserPermissionsLogic(context.Background(), ctx)
	deptId := int64(3)
	resp, err := logic.GetUserPermissions(&types.GetUserPermissionsReq{Id: 7, DeptId: &deptId})
	if err != nil || !resp.Items[0].UiGranted || !resp.Items[0].ApiAllowed || len(resp.VisibleUsers) != 1 || resp.VisibleUserCount != 1 {
		t.Fatalf("部门内应显示菜单、接口和可见账号: %+v; %v", resp, err)
	}
	deptId = 4
	resp, err = logic.GetUserPermissions(&types.GetUserPermissionsReq{Id: 7, DeptId: &deptId})
	if err != nil || !resp.Items[0].UiGranted || resp.Items[0].ApiAllowed || len(resp.VisibleUsers) != 0 || resp.VisibleUserCount != 0 {
		t.Fatalf("接口禁止时不能返回可见账号: %+v; %v", resp, err)
	}
}
