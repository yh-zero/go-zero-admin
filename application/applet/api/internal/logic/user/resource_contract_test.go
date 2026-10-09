package user

import (
	"context"
	"encoding/json"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	userclient "go-zero-admin/application/applet/rpc/client/user"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
	"testing"
)

type resourceClient struct {
	userclient.User
	preview  *pb.UserResourcePreviewRequest
	transfer *pb.TransferUserResourcesRequest
}

func (c *resourceClient) GetUserResourcePreview(ctx context.Context, in *pb.UserResourcePreviewRequest, opts ...grpc.CallOption) (*pb.UserResourcePreviewResponse, error) {
	c.preview = in
	return &pb.UserResourcePreviewResponse{UserID: in.UserID, Files: 3, AIAvailable: false, AIReason: "different database", TransferMode: "files_only"}, nil
}
func (c *resourceClient) TransferUserResources(ctx context.Context, in *pb.TransferUserResourcesRequest, opts ...grpc.CallOption) (*pb.TransferUserResourcesResponse, error) {
	c.transfer = in
	return &pb.TransferUserResourcesResponse{Files: 3, TransactionMode: "same_database"}, nil
}
func TestResourceAPIBindsAuthenticatedActorAndResult(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]interface{}{"ID": json.Number("11"), "AuthorityId": json.Number("801"), "SessionVersion": json.Number("5"), "SessionID": "device"})
	client := &resourceClient{}
	s := &svc.ServiceContext{AppletUserRPC: client}
	result, err := NewGetUserResourcePreviewLogic(ctx, s).GetUserResourcePreview(&types.UserResourcePreviewRequest{UserID: 7})
	if err != nil || result == nil || result.UserID != 7 || result.Files != 3 || result.AIReason != "different database" {
		t.Fatalf("preview must forward explicit unavailable/result: %+v %v", result, err)
	}
	if client.preview == nil || client.preview.Actor.UserID != 11 || client.preview.Actor.SessionVersion != 5 || client.preview.Actor.SessionID != "device" {
		t.Fatal("preview did not bind authenticated actor")
	}
	moved, err := NewTransferUserResourcesLogic(ctx, s).TransferUserResources(&types.TransferUserResourcesRequest{UserID: 7, TargetUserID: 8, IncludeFiles: true})
	if err != nil || moved == nil || moved.Files != 3 || moved.TransactionMode != "same_database" {
		t.Fatalf("transfer result lost: %+v %v", moved, err)
	}
	if client.transfer == nil || client.transfer.Actor.UserID != 11 || client.transfer.UserID != 7 || client.transfer.TargetUserID != 8 || !client.transfer.IncludeFiles || client.transfer.IncludeAI {
		t.Fatal("transfer explicit choices/actor lost")
	}
}
