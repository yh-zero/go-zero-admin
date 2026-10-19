package user

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	userclient "go-zero-admin/application/applet/rpc/client/user"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc"
)

type registerEmailClient struct {
	userclient.User
	request *pb.RegisterRequest
}

func (c *registerEmailClient) Register(_ context.Context, req *pb.RegisterRequest, _ ...grpc.CallOption) (*pb.RegisterResponse, error) {
	c.request = req
	return &pb.RegisterResponse{}, nil
}

func TestAdminRegistrationEmailVerificationRemainsOptional(t *testing.T) {
	for _, tc := range []struct{ fields, code, scene string }{
		{},
		{`,"emailCode":"123456","scene":"register"`, "123456", "register"},
	} {
		r := httptest.NewRequest("POST", "/v1/sys/register", strings.NewReader(`{"userName":"admin-created","passWord":"test-password","nickName":"member","authorityId":2,"authorityIds":[2],"email":"member@example.test"`+tc.fields+`}`))
		r.Header.Set("Content-Type", "application/json")
		var req types.RegisterRequest
		if err := httpx.Parse(r, &req); err != nil {
			t.Fatal(err)
		}
		client := &registerEmailClient{}
		if _, err := NewRegisterLogic(r.Context(), &svc.ServiceContext{AppletUserRPC: client}).Register(&req); err != nil {
			t.Fatal(err)
		}
		got := client.request
		if got.EmailCode != tc.code || got.Scene != tc.scene || got.UserInfo.Email != "member@example.test" || len(got.AuthorityIds) != 1 || got.AuthorityIds[0] != 2 {
			t.Fatalf("existing registration contract changed: %+v", got)
		}
	}
}
