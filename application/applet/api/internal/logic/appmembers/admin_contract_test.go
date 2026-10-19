package appmembers

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/appaccountservice"
	"go-zero-admin/application/applet/rpc/client/appmembershipservice"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc"
)

type accountAdminClient struct {
	appaccountservice.AppAccountService
	request *pb.AppAdminListRequest
	err     error
}

func (c *accountAdminClient) ListAppUsers(_ context.Context, req *pb.AppAdminListRequest, _ ...grpc.CallOption) (*pb.AppUserListResponse, error) {
	c.request = req
	return &pb.AppUserListResponse{}, c.err
}

type membershipAdminClient struct {
	appmembershipservice.AppMembershipService
	request *pb.AppGrantRequest
	err     error
}

func (c *membershipAdminClient) GrantAppMembership(_ context.Context, req *pb.AppGrantRequest, _ ...grpc.CallOption) (*pb.AppGrantInfo, error) {
	c.request = req
	return &pb.AppGrantInfo{ID: 10, UserID: req.UserID, PlanID: req.PlanID, PlanVersion: 3, Entitlements: []string{"profile.export"}, Status: "scheduled"}, c.err
}
func adminContext() context.Context {
	return context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]any{"ID": 7, "AuthorityId": 9, "SessionID": "admin-session", "SessionVersion": 3})
}
func assertActor(t *testing.T, actor *pb.SessionRequest) {
	t.Helper()
	if actor == nil || actor.UserID != 7 || actor.AuthorityId != 9 || actor.SessionID != "admin-session" || actor.SessionVersion != 3 {
		t.Fatalf("administrator identity not derived from authenticated context: %+v", actor)
	}
}
func TestListUsersClampsPaginationAndPreservesAdminSession(t *testing.T) {
	for _, tc := range []struct{ no, size, wantNo, wantSize int64 }{{0, 0, 1, 10}, {-3, 1000, 1, 100}, {2, 25, 2, 25}} {
		client := &accountAdminClient{}
		result, err := NewListAppUsersLogic(adminContext(), &svc.ServiceContext{AppAccountRPC: client}).ListAppUsers(&types.AppUserListRequest{PageNo: tc.no, PageSize: tc.size, Keyword: "member", Status: "pending"})
		if err != nil {
			t.Fatal(err)
		}
		got := client.request
		assertActor(t, got.Actor)
		if got.Status != "pending" || got.Page.Keyword != "member" || got.Page.PageNo != tc.wantNo || got.Page.PageSize != tc.wantSize {
			t.Fatalf("list filters changed: %+v", got)
		}
		if result.List == nil || result.PageNo != tc.wantNo || result.PageSize != tc.wantSize {
			t.Fatalf("invalid empty-page response: %+v", result)
		}
	}
}
func TestGrantPreservesIdempotencyAndIgnoresBodyActor(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/sys/app-members/grants", strings.NewReader(`{"userId":41,"planId":2,"days":30,"requestId":"stable-operation-id","reason":"人工续期","actor":{"userId":99,"authorityId":1}}`))
	r.Header.Set("Content-Type", "application/json")
	var input types.AppGrantRequest
	if err := httpx.Parse(r, &input); err != nil {
		t.Fatal(err)
	}
	client := &membershipAdminClient{}
	result, err := NewGrantAppMembershipLogic(adminContext(), &svc.ServiceContext{AppMembershipRPC: client}).GrantAppMembership(&input)
	if err != nil {
		t.Fatal(err)
	}
	got := client.request
	assertActor(t, got.Actor)
	if got.UserID != 41 || got.PlanID != 2 || got.Days != 30 || got.RequestID != "stable-operation-id" || got.Reason != "人工续期" {
		t.Fatalf("grant contract changed: %+v", got)
	}
	if result.UserID != 41 || result.PlanID != 2 || result.PlanVersion != 3 || result.Status != "scheduled" || len(result.Entitlements) != 1 {
		t.Fatalf("grant snapshot changed: %+v", result)
	}
}
func TestAdminRPCPermissionErrorsPropagate(t *testing.T) {
	want := errors.New("permission denied")
	users, err := NewListAppUsersLogic(adminContext(), &svc.ServiceContext{AppAccountRPC: &accountAdminClient{err: want}}).ListAppUsers(&types.AppUserListRequest{})
	if users != nil || !errors.Is(err, want) {
		t.Fatalf("list denial lost: result=%+v error=%v", users, err)
	}
	grant, err := NewGrantAppMembershipLogic(adminContext(), &svc.ServiceContext{AppMembershipRPC: &membershipAdminClient{err: want}}).GrantAppMembership(&types.AppGrantRequest{})
	if grant != nil || !errors.Is(err, want) {
		t.Fatalf("grant denial lost: result=%+v error=%v", grant, err)
	}
}
