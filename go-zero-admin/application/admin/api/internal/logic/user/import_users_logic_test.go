package user

import (
	"context"
	"testing"

	"google.golang.org/grpc"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/client/userservice"
	"go-zero-admin/pkg/casbinauth"
	"go-zero-admin/pkg/xerr"
)

type importUsersClient struct {
	userservice.UserService
	request *adminpb.ImportUsersRequest
}

func (c *importUsersClient) ImportUsers(_ context.Context, req *adminpb.ImportUsersRequest, _ ...grpc.CallOption) (*adminpb.ImportUsersResponse, error) {
	c.request = req
	return &adminpb.ImportUsersResponse{
		Created: 1,
		Rows:    []*adminpb.ImportUserResult{{Row: 2, Username: "alice", Status: "created"}},
	}, nil
}

func TestImportUsersRequiresUserCreatePermission(t *testing.T) {
	holder, err := casbinauth.NewHolder([]casbinauth.PolicyRule{
		{Ptype: "g", V0: "user:8", V1: "manager", V2: "dept:12"},
		{Ptype: "p", V0: "manager", V1: "dept:12", V2: "/v1/sys/user/import", V3: "POST"},
		{Ptype: "p", V0: "manager", V1: "dept:34", V2: "/v1/sys/user/create", V3: "POST"},
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &importUsersClient{}
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUserID, int64(8))
	ctx = context.WithValue(ctx, middleware.CtxKeyDeptId, int64(12))
	logic := NewImportUsersLogic(ctx, &svc.ServiceContext{Validator: svc.NewValidator(), UserRpc: client, Enforcer: holder})
	_, err = logic.ImportUsers(&types.ImportUsersReq{Csv: "username,dept_id,real_name\nalice,12,\n"})
	if code, ok := xerr.AsCodeError(err); !ok || code.GetErrCode() != xerr.PERMISSION_DENIED || client.request != nil {
		t.Fatalf("仅有导入操作权限时不能查询或写入: %v", err)
	}
}

func TestImportUsersConfirmKeepsExplicitSelection(t *testing.T) {
	holder, err := casbinauth.NewHolder([]casbinauth.PolicyRule{
		{Ptype: "g", V0: "user:8", V1: "manager", V2: "dept:12"},
		{Ptype: "p", V0: "manager", V1: "dept:12", V2: "/v1/sys/user/create", V3: "POST"},
	})
	if err != nil {
		t.Fatal(err)
	}
	client := &importUsersClient{}
	ctx := context.WithValue(context.Background(), middleware.CtxKeyUserID, int64(8))
	ctx = context.WithValue(ctx, middleware.CtxKeyDeptId, int64(12))
	logic := NewImportUsersLogic(ctx, &svc.ServiceContext{Validator: svc.NewValidator(), UserRpc: client, Enforcer: holder})
	csv := "username,dept_id,real_name\nalice,12,\nsecond,12,\nbad,invalid,\n"
	resp, err := logic.ImportUsers(&types.ImportUsersReq{Csv: csv, Confirm: true, RowNumbers: []int64{2, 4}})
	if err != nil || resp.Created != 1 || len(resp.Rows) != 2 || resp.Rows[1].Row != 4 || resp.Rows[1].Status != "invalid" {
		t.Fatalf("确认结果应逐行返回选择范围内的成功与格式错误: %+v, %v", resp, err)
	}
	if sent := client.request; sent == nil || len(sent.Rows) != 2 || len(sent.RowNumbers) != 1 || sent.RowNumbers[0] != 2 || sent.Actor.GetDeptId() != 12 {
		t.Fatalf("应保留选择的有效行以及用于检查重复的所有格式正确行: %v", sent)
	}
	client.request = nil
	_, err = logic.ImportUsers(&types.ImportUsersReq{Csv: csv, Confirm: true, RowNumbers: []int64{9}})
	if code, ok := xerr.AsCodeError(err); !ok || code.GetErrCode() != xerr.REQUEST_PARAM_ERROR || client.request != nil {
		t.Fatalf("未知行号应在调用导入前拒绝: %v", err)
	}
}
