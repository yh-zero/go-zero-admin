package base

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/fileresourceservice"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
)

func uploadRequest(t *testing.T, field string, data []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile(field, "example.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = part.Write(data); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func TestImageUploadValidation(t *testing.T) {
	var valid bytes.Buffer
	if err := png.Encode(&valid, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	// readImage is still a multipart parser in the API layer; it should accept
	// a real PNG and reject the obvious abuse vectors before the bytes ever
	// leave the process.
	if _, mime, err := readImage(uploadRequest(t, "file_img", valid.Bytes())); err != nil || mime != "image/png" {
		t.Fatalf("valid PNG rejected: %v", err)
	}
	for _, tc := range []struct {
		name, field string
		data        []byte
	}{
		{"wrong field", "file", valid.Bytes()},
		{"empty", "file_img", nil},
		{"HTML disguised as PNG", "file_img", []byte("<html><script>alert(1)</script></html>")},
		{"too large", "file_img", bytes.Repeat([]byte("x"), maxFileSize+1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, _, err := readImage(uploadRequest(t, tc.field, tc.data)); err == nil {
				t.Fatal("invalid upload accepted")
			}
		})
	}
}

// capturingFileClient records the request UploadFile receives, so we can
// assert that the API layer forwards bytes, mime, filename, visibility,
// and the JWT-derived actor.
type capturingFileClient struct {
	fileresourceservice.FileResourceService
	request *pb.UploadFileRequest
}

func (c *capturingFileClient) UploadFile(ctx context.Context, req *pb.UploadFileRequest, _ ...grpc.CallOption) (*pb.UploadFileResponse, error) {
	c.request = req
	return &pb.UploadFileResponse{File: &pb.FileResource{ID: 99, ObjectKey: "go-zero-admin/abc.png"}, Url: "https://returned/abc.png"}, nil
}

func TestUploadFileImgForwardsBytesAndActorToRPC(t *testing.T) {
	var valid bytes.Buffer
	if err := png.Encode(&valid, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	rpc := &capturingFileClient{}
	s := &svc.ServiceContext{AppletFileRPC: rpc}
	req := uploadRequest(t, "file_img", valid.Bytes())
	req = req.WithContext(ctxWithJwt(req.Context()))
	resp, err := NewUploadFileImgLogic(req.Context(), s).UploadFileImg(&types.UploadFileImgRequest{}, req)
	if err != nil {
		t.Fatal(err)
	}
	if rpc.request == nil || len(rpc.request.Data) == 0 || rpc.request.Mime != "image/png" || rpc.request.Visibility != "public" || rpc.request.Actor == nil || rpc.request.Actor.UserID != 10 || resp.FileId != 99 {
		t.Fatalf("forwarded request unexpected: %+v", rpc.request)
	}
}

func TestUploadFileImgRejectsWhenFileRPCMissing(t *testing.T) {
	var valid bytes.Buffer
	if err := png.Encode(&valid, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	req := uploadRequest(t, "file_img", valid.Bytes())
	req = req.WithContext(ctxWithJwt(req.Context()))
	s := &svc.ServiceContext{}
	if _, err := NewUploadFileImgLogic(req.Context(), s).UploadFileImg(&types.UploadFileImgRequest{}, req); err == nil {
		t.Fatal("missing RPC accepted")
	}
}

func ctxWithJwt(ctx context.Context) context.Context {
	return context.WithValue(ctx, ctxJwt.CtxKeyJwtData, map[string]any{"ID": 10, "AuthorityId": 2, "SessionVersion": 1, "SessionID": "api-test"})
}
