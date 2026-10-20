package svc_test

import (
	"bytes"
	"context"
	"mime/multipart"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	baseLogic "go-zero-admin/application/applet/api/internal/logic/base"
	filesLogic "go-zero-admin/application/applet/api/internal/logic/files"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/fileresourceservice"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"

	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

type slowFileServer struct {
	pb.UnimplementedFileResourceServiceServer
}

func waitForFileStorage(ctx context.Context) error {
	timer := time.NewTimer(2200 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (slowFileServer) UploadFile(ctx context.Context, in *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	if err := waitForFileStorage(ctx); err != nil {
		return nil, err
	}
	return &pb.UploadFileResponse{File: &pb.FileResource{ID: 7}, Url: "http://storage/large.png"}, nil
}

func (slowFileServer) DeleteFile(ctx context.Context, in *pb.FileIDRequest) (*pb.NoDataResponse, error) {
	if err := waitForFileStorage(ctx); err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, nil
}

// Use go-zero's real two-second client timeout middleware. The API must allow
// storage operations to complete past that deadline without changing the
// shared client's deadline for unrelated business methods.
func TestFileAPIOperationsOutliveDefaultRPCTimeout(t *testing.T) {
	listener := bufconn.Listen(1 << 20)
	server := grpc.NewServer()
	pb.RegisterFileResourceServiceServer(server, slowFileServer{})
	go func() { _ = server.Serve(listener) }()
	defer server.Stop()
	client, err := zrpc.NewClient(zrpc.RpcClientConf{Target: "passthrough:///bufnet", NonBlock: true, Timeout: 2000, Middlewares: zrpc.ClientMiddlewaresConf{Timeout: true}}, zrpc.WithDialOption(grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) })))
	if err != nil {
		t.Fatal(err)
	}
	defer client.Conn().Close()
	service := &svc.ServiceContext{AppletFileRPC: fileresourceservice.NewFileResourceService(client)}
	ctx := context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]any{"ID": 10, "AuthorityId": 2, "SessionVersion": 1, "SessionID": "file-test"})
	t.Run("upload", func(t *testing.T) {
		var body bytes.Buffer
		writer := multipart.NewWriter(&body)
		part, err := writer.CreateFormFile("file_img", "image.png")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = part.Write([]byte("\x89PNG\r\n\x1a\n"))
		if err := writer.Close(); err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("POST", "/uploadFileImg", &body).WithContext(ctx)
		request.Header.Set("Content-Type", writer.FormDataContentType())
		response, err := baseLogic.NewUploadFileImgLogic(ctx, service).UploadFileImg(&types.UploadFileImgRequest{}, request)
		if err != nil || response.FileId != 7 {
			t.Fatalf("slow storage upload did not complete: %+v %v", response, err)
		}
	})
	t.Run("delete", func(t *testing.T) {
		response, err := filesLogic.NewDeleteFileLogic(ctx, service).DeleteFile(&types.FileIDRequest{Id: 7})
		if err != nil || response.Message == "" {
			t.Fatalf("slow storage deletion did not complete: %+v %v", response, err)
		}
	})
}
