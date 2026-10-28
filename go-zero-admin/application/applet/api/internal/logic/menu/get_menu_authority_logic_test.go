package menu

import (
	"context"
	"encoding/json"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	menurpc "go-zero-admin/application/applet/rpc/client/menu"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
)

type menuRPCStub struct {
	menurpc.Menu
	role int64
}

func (m *menuRPCStub) GetMenuAuthority(_ context.Context, in *pb.GetMenuAuthorityRequest, _ ...grpc.CallOption) (*pb.GetMenuAuthorityResponse, error) {
	m.role = in.AuthorityId
	return &pb.GetMenuAuthorityResponse{SysMenuList: []*pb.SysMenu{{
		ID: 5, MenuId: "5", AuthorityId: in.AuthorityId,
		SysBaseMenu: &pb.SysBaseMenu{ID: 5, Name: "user", Path: "user", Meta: &pb.Meta{Title: "用户管理"}, MenuBtn: []*pb.SysBaseMenuBtn{{ID: 12, Name: "create", SysBaseMenuID: 5}}},
	}}}, nil
}

// Protect the HTTP/RPC contract: target role is the query role, and nested RPC
// menu data must survive copying into the flattened HTTP response.
func TestGetMenuAuthorityUsesTargetRoleAndPreservesDefinitions(t *testing.T) {
	client := &menuRPCStub{}
	ctx := context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]interface{}{"AuthorityId": json.Number("88")})
	result, err := NewGetMenuAuthorityLogic(ctx, &svc.ServiceContext{AppletMenuRPC: client}).GetMenuAuthority(&types.GetMenuAuthorityRequest{AuthorityId: 99})
	if err != nil {
		t.Fatal(err)
	}
	if client.role != 99 {
		t.Fatalf("queried role %d, expected target role 99", client.role)
	}
	if len(result.SysMenuList) != 1 {
		t.Fatalf("lost menu list: %+v", result)
	}
	menu := result.SysMenuList[0]
	if menu.ID != 5 || menu.Name != "user" || menu.Meta.Title != "用户管理" || len(menu.MenuBtn) != 1 || menu.MenuBtn[0].ID != 12 {
		t.Fatalf("lost nested menu/button fields: %+v", menu)
	}
}
