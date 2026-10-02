package base

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/fileresourceservice"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
)

type uploadResourceClient struct {
	fileresourceservice.FileResourceService
	requests []*pb.RegisterFileRequest
	failures int
}

func (c *uploadResourceClient) RegisterFile(ctx context.Context, req *pb.RegisterFileRequest, _ ...grpc.CallOption) (*pb.FileResource, error) {
	c.requests = append(c.requests, req)
	if len(c.requests) <= c.failures {
		return nil, errors.New("temporary transport error")
	}
	return &pb.FileResource{ID: 44, ObjectKey: req.File.ObjectKey}, nil
}

func resourceUploadRequest(t *testing.T, name, visibility string) *http.Request {
	t.Helper()
	var content bytes.Buffer
	if err := png.Encode(&content, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file_img", name)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write(content.Bytes())
	if visibility != "" {
		_ = writer.WriteField("visibility", visibility)
	}
	_ = writer.Close()
	request := httptest.NewRequest(http.MethodPost, "/v1/sys/upload", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request.WithContext(context.WithValue(request.Context(), ctxJwt.CtxKeyJwtData, map[string]any{"ID": 10, "AuthorityId": 2, "SessionVersion": 1, "SessionID": "upload-device"}))
}

func TestPrivateImageUploadRegistersOwnershipAndRetriesSameObject(t *testing.T) {
	var acl string
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.Method != http.MethodPut {
			t.Errorf("upload issued unsafe %s", r.Method)
		}
		acl = r.Header.Get("X-Oss-Object-Acl")
		content, _ := io.ReadAll(r.Body)
		if http.DetectContentType(content) != "image/png" {
			t.Error("upload body changed")
		}
		w.Header().Set("ETag", `"mock-etag"`)
		w.WriteHeader(200)
	}))
	defer server.Close()
	client, err := oss.New(server.URL, "test-access-key", "test-secret", oss.UseCname(true))
	if err != nil {
		t.Fatal(err)
	}
	rpc := &uploadResourceClient{failures: 1}
	s := &svc.ServiceContext{OssClient: client, AppletFileRPC: rpc}
	s.Config.Oss.BucketName = "test-bucket"
	s.Config.Oss.Endpoint = server.URL
	r := resourceUploadRequest(t, "example.png", "private")
	response, err := NewUploadFileImgLogic(r.Context(), s).UploadFileImg(&types.UploadFileImgRequest{}, r)
	if err != nil {
		t.Fatal(err)
	}
	if acl != "private" || response.FileId != 44 || !strings.Contains(response.FileImgUrl, "Signature=") || requests != 1 || len(rpc.requests) != 2 {
		t.Fatalf("upload registration/ACL unexpected acl=%s file=%d uploads=%d register=%d", acl, response.FileId, requests, len(rpc.requests))
	}
	first, second := rpc.requests[0], rpc.requests[1]
	if first.File.ObjectKey != second.File.ObjectKey || first.Actor.UserID != 10 || first.Actor.SessionID != "upload-device" || first.File.Name != "example.png" || first.File.Visibility != "private" {
		t.Fatalf("registration did not preserve identity or key: %+v", first)
	}
}

func TestInvalidUploadFilenameIsRejectedBeforeOSS(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { requests++; w.WriteHeader(200) }))
	defer server.Close()
	client, err := oss.New(server.URL, "test-key", "test-secret", oss.UseCname(true))
	if err != nil {
		t.Fatal(err)
	}
	s := &svc.ServiceContext{OssClient: client, AppletFileRPC: &uploadResourceClient{}}
	s.Config.Oss.BucketName = "test-bucket"
	r := resourceUploadRequest(t, strings.Repeat("x", 256)+".png", "")
	if _, err := NewUploadFileImgLogic(r.Context(), s).UploadFileImg(&types.UploadFileImgRequest{}, r); err == nil || requests != 0 {
		t.Fatal("invalid filename wrote an orphan OSS object")
	}
}
