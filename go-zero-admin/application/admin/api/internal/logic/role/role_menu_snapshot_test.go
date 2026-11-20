package role

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/client/roleservice"
	"go-zero-admin/pkg/xerr"
	"google.golang.org/grpc"
)

type captureRoleWrites struct {
	roleservice.RoleService
	menus *adminpb.UpdateRoleMenusRequest
	role  *adminpb.UpdateRoleRequest
}

func (r *captureRoleWrites) UpdateRoleMenus(_ context.Context, in *adminpb.UpdateRoleMenusRequest, _ ...grpc.CallOption) (*adminpb.NoDataResponse, error) {
	r.menus = in
	return &adminpb.NoDataResponse{}, nil
}

func (r *captureRoleWrites) UpdateRole(_ context.Context, in *adminpb.UpdateRoleRequest, _ ...grpc.CallOption) (*adminpb.NoDataResponse, error) {
	r.role = in
	return &adminpb.NoDataResponse{}, nil
}

func TestRoleMenuSnapshotPresence(t *testing.T) {
	for _, tc := range []struct {
		name, method, body string
		rejected, snapshot bool
	}{
		{"独立授权缺原集合", "POST", `{"menu_ids":[]}`, true, false},
		{"独立授权原集合为null", "POST", `{"expected_menu_ids":null,"menu_ids":[]}`, true, false},
		{"不可省略目标集合", "POST", `{"expected_menu_ids":[]}`, true, false},
		{"空原集合与空目标合法", "POST", `{"expected_menu_ids":[],"menu_ids":[]}`, false, true},
		{"角色资料不需原集合", "PUT", `{"name":"新角色名","menu_ids":null}`, false, false},
		{"普通编辑清空也需原集合", "PUT", `{"menu_ids":[]}`, true, false},
		{"普通编辑接受空原集合", "PUT", `{"menu_ids":[3],"expected_menu_ids":[]}`, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, "/", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			recorder := &captureRoleWrites{}
			service := &svc.ServiceContext{RoleRpc: recorder, Validator: svc.NewValidator()}
			var err error
			if tc.method == "POST" {
				input := types.UpdateRoleMenusReq{Id: 1}
				if err = httpx.ParseJsonBody(req, &input); err == nil {
					_, err = NewUpdateRoleMenusLogic(req.Context(), service).UpdateRoleMenus(&input)
				}
			} else {
				input := types.UpdateRoleReq{Id: 1}
				if err = httpx.ParseJsonBody(req, &input); err == nil {
					_, err = NewUpdateRoleLogic(req.Context(), service).UpdateRole(&input)
				}
			}
			if tc.rejected {
				if err == nil || recorder.menus != nil || recorder.role != nil {
					t.Fatalf("不完整的请求应在调用 RPC 前拒绝，得到 %v", err)
				}
				_, body := xerr.Handler(req.Context(), err)
				if body.(map[string]any)["code"] != uint32(400) {
					t.Fatalf("应返回业务码 400，得到 %v", body)
				}
				return
			}
			if err != nil || recorder.role == nil && recorder.menus == nil {
				t.Fatalf("请求应成功传给 RPC，得到 %v", err)
			}
			got := recorder.menus.GetExpectedMenus() != nil || recorder.role.GetExpectedMenus() != nil
			if got != tc.snapshot || recorder.role != nil && recorder.role.UpdateMenus != tc.snapshot {
				t.Fatal("原集合缺失与显式空集合在跨端转换中被混淆")
			}
		})
	}
}
