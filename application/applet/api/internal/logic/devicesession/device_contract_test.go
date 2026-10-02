package devicesession

import (
	"context"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
	"testing"
)

type deviceClient struct {
	list   *pb.DeviceSessionListRequest
	revoke *pb.RevokeDeviceSessionRequest
}

func (c *deviceClient) GetDeviceSessions(_ context.Context, in *pb.DeviceSessionListRequest, _ ...grpc.CallOption) (*pb.DeviceSessionListResponse, error) {
	c.list = in
	return &pb.DeviceSessionListResponse{Total: 1, List: []*pb.DeviceSession{{ID: "test-device", UserID: in.Actor.UserID, Username: "tester", Current: true}}}, nil
}
func (c *deviceClient) RevokeDeviceSession(_ context.Context, in *pb.RevokeDeviceSessionRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	c.revoke = in
	return &pb.NoDataResponse{}, nil
}
func TestPersonalDeviceEndpointsKeepOwnershipAndDeviceIdentity(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]interface{}{"ID": int64(9), "AuthorityId": int64(1), "SessionVersion": int64(4), "SessionID": "signed-device-id"})
	client := &deviceClient{}
	s := &svc.ServiceContext{AppletSessionManageRPC: client}
	s.Config.Page.PageNo = 1
	s.Config.Page.PageSize = 20
	response, err := NewGetMyDeviceSessionsLogic(ctx, s).GetMyDeviceSessions(&types.DeviceSessionListRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if client.list.UserID != 9 || client.list.Actor.SessionID != "signed-device-id" || client.list.PageRequest.PageSize != 20 || response.PageNo != 1 || !response.List[0].Current {
		t.Fatal("personal query identity/defaults changed", client.list, response)
	}
	if _, err := NewRevokeMyDeviceSessionLogic(ctx, s).RevokeMyDeviceSession(&types.RevokeDeviceSessionRequest{Id: "target"}); err != nil {
		t.Fatal(err)
	}
	if !client.revoke.SelfOnly || client.revoke.Actor.UserID != 9 || client.revoke.Actor.SessionVersion != 4 {
		t.Fatal("personal admin revocation lost ownership restriction", client.revoke)
	}
	if _, err := NewRevokeDeviceSessionLogic(ctx, s).RevokeDeviceSession(&types.RevokeDeviceSessionRequest{Id: "target"}); err != nil {
		t.Fatal(err)
	}
	if client.revoke.SelfOnly {
		t.Fatal("admin revocation incorrectly uses personal restriction")
	}
}
