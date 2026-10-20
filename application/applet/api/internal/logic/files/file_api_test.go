package files

import (
	"context"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/fileresourceservice"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
)

type capturingFileService struct {
	fileresourceservice.FileResourceService
	getURL    *pb.FileIDRequest
	deleteReq *pb.FileIDRequest
}

func (c *capturingFileService) GetFileURL(ctx context.Context, req *pb.FileIDRequest, _ ...grpc.CallOption) (*pb.FileURLResponse, error) {
	c.getURL = req
	return &pb.FileURLResponse{Url: "https://signed.example/object", ExpiresIn: 300}, nil
}

func (c *capturingFileService) DeleteFile(ctx context.Context, req *pb.FileIDRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	c.deleteReq = req
	return &pb.NoDataResponse{}, nil
}

func fileAPIContext() context.Context {
	return context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]any{"ID": 10, "AuthorityId": 2, "SessionVersion": 1, "SessionID": "device-session"})
}

// The API tier no longer talks to object storage directly; the RPC owns signing and
// deletion. These tests pin the pass-through contract: actor session and
// file ID must be forwarded unchanged.
func TestGetFileURLForwardsActorAndID(t *testing.T) {
	rpc := &capturingFileService{}
	s := &svc.ServiceContext{AppletFileRPC: rpc}
	resp, err := NewGetFileURLLogic(fileAPIContext(), s).GetFileURL(&types.FileURLRequest{Id: 4})
	if err != nil {
		t.Fatal(err)
	}
	if rpc.getURL == nil || rpc.getURL.ID != 4 || rpc.getURL.Actor == nil || rpc.getURL.Actor.SessionID != "device-session" {
		t.Fatalf("request not forwarded: %+v", rpc.getURL)
	}
	if resp.Url == "" || resp.ExpiresIn != 300 {
		t.Fatalf("response not mapped: %+v", resp)
	}
}

func TestDeleteFileForwardsActorAndID(t *testing.T) {
	rpc := &capturingFileService{}
	s := &svc.ServiceContext{AppletFileRPC: rpc}
	resp, err := NewDeleteFileLogic(fileAPIContext(), s).DeleteFile(&types.FileIDRequest{Id: 7})
	if err != nil {
		t.Fatal(err)
	}
	if rpc.deleteReq == nil || rpc.deleteReq.ID != 7 || rpc.deleteReq.Actor == nil || rpc.deleteReq.Actor.UserID != 10 {
		t.Fatalf("request not forwarded: %+v", rpc.deleteReq)
	}
	if resp.Message == "" {
		t.Fatal("expected confirmation message")
	}
}
