package appmembers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/appaccountservice"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc"
)

type avatarAdminClient struct {
	appaccountservice.AppAccountService
	request *pb.AppAvatarRequest
	err     error
}

func (c *avatarAdminClient) SetAppUserAvatar(_ context.Context, req *pb.AppAvatarRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	c.request = req
	return &pb.NoDataResponse{}, c.err
}

func (c *avatarAdminClient) ListAppUsers(context.Context, *pb.AppAdminListRequest, ...grpc.CallOption) (*pb.AppUserListResponse, error) {
	return &pb.AppUserListResponse{List: []*pb.AppUserInfo{{ID: 41, AvatarURL: "https://files.example.test/avatar.png"}}, Total: 1}, nil
}

func TestAppUserListExposesAvatarUsingTheAppJSONContract(t *testing.T) {
	response, err := NewListAppUsersLogic(adminContext(), &svc.ServiceContext{AppAccountRPC: &avatarAdminClient{}}).ListAppUsers(&types.AppUserListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	var decoded struct {
		List []struct {
			AvatarURL string `json:"avatarUrl"`
		} `json:"list"`
	}
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.List) != 1 || decoded.List[0].AvatarURL != "https://files.example.test/avatar.png" {
		t.Fatalf("App HTTP list lost avatarUrl: %s", encoded)
	}
}

func TestAppAvatarRequiresURLPresenceAndForwardsClearWithAuthenticatedActor(t *testing.T) {
	for _, next := range []string{"", "https://files.example.test/gozero-public/go-zero-admin/avatar.png"} {
		request := httptest.NewRequest("PUT", "/v1/sys/app-members/avatar", strings.NewReader(`{"userId":41,"avatarUrl":"`+next+`","actor":{"userId":999,"authorityId":1}}`))
		request.Header.Set("Content-Type", "application/json")
		var input types.AppAvatarRequest
		if err := httpx.Parse(request, &input); err != nil {
			t.Fatal(err)
		}
		client := &avatarAdminClient{}
		response, err := NewSetAppUserAvatarLogic(adminContext(), &svc.ServiceContext{AppAccountRPC: client}).SetAppUserAvatar(&input)
		if err != nil || response == nil {
			t.Fatalf("avatar change failed: %+v %v", response, err)
		}
		if client.request == nil {
			t.Fatal("avatar request was not forwarded")
		}
		assertActor(t, client.request.Actor)
		if client.request.UserID != 41 || client.request.AvatarURL != next {
			t.Fatalf("avatar contract changed: %+v", client.request)
		}
	}
	request := httptest.NewRequest("PUT", "/v1/sys/app-members/avatar", strings.NewReader(`{"userId":41}`))
	request.Header.Set("Content-Type", "application/json")
	if err := httpx.Parse(request, &types.AppAvatarRequest{}); err == nil {
		t.Fatal("omitting avatarUrl must not silently clear the avatar")
	}
}

func TestAppAvatarAuthorizationErrorPropagates(t *testing.T) {
	want := errors.New("permission denied")
	client := &avatarAdminClient{err: want}
	response, err := NewSetAppUserAvatarLogic(adminContext(), &svc.ServiceContext{AppAccountRPC: client}).SetAppUserAvatar(&types.AppAvatarRequest{UserID: 41, AvatarURL: ""})
	if response != nil || !errors.Is(err, want) {
		t.Fatalf("avatar permission denial lost: %+v %v", response, err)
	}
}
