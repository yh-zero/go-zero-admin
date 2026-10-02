package files

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/fileresourceservice"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
)

type fileAPIClient struct {
	fileresourceservice.FileResourceService
	file                    *pb.FileResource
	beginCalls, finishCalls int
	actor                   *pb.SessionRequest
}

func (c *fileAPIClient) GetFile(ctx context.Context, req *pb.FileIDRequest, _ ...grpc.CallOption) (*pb.FileResource, error) {
	c.actor = req.Actor
	return c.file, nil
}
func (c *fileAPIClient) BeginDeleteFile(ctx context.Context, req *pb.FileIDRequest, _ ...grpc.CallOption) (*pb.FileResource, error) {
	c.beginCalls++
	c.actor = req.Actor
	return c.file, nil
}
func (c *fileAPIClient) FinishDeleteFile(ctx context.Context, req *pb.FileIDRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	c.finishCalls++
	c.file.Status = "deleted"
	return &pb.NoDataResponse{}, nil
}

func fileAPIContext() context.Context {
	return context.WithValue(context.Background(), ctxJwt.CtxKeyJwtData, map[string]any{"ID": 10, "AuthorityId": 2, "SessionVersion": 1, "SessionID": "device-session"})
}
func fileOSS(t *testing.T, handler http.HandlerFunc) (*svc.ServiceContext, *httptest.Server) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := oss.New(server.URL, "test-access-key", "test-secret", oss.UseCname(true))
	if err != nil {
		t.Fatal(err)
	}
	s := &svc.ServiceContext{OssClient: client}
	s.Config.Oss.BucketName = "test-bucket"
	s.Config.Oss.Endpoint = server.URL
	return s, server
}

func TestPrivateFileURLIsShortLivedAndUsesActorSession(t *testing.T) {
	s, _ := fileOSS(t, func(w http.ResponseWriter, r *http.Request) { t.Fatal("signing URL must not access network") })
	client := &fileAPIClient{file: &pb.FileResource{ID: 4, ObjectKey: "go-zero-admin/test.png", Visibility: "private", Status: "active"}}
	s.AppletFileRPC = client
	response, err := NewGetFileURLLogic(fileAPIContext(), s).GetFileURL(&types.FileURLRequest{Id: 4})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(response.Url)
	if err != nil {
		t.Fatal(err)
	}
	expiry, _ := strconv.ParseInt(parsed.Query().Get("Expires"), 10, 64)
	remaining := expiry - time.Now().Unix()
	if response.ExpiresIn != 300 || remaining < 298 || remaining > 301 || parsed.Query().Get("Signature") == "" || client.actor.SessionID != "device-session" {
		t.Fatalf("invalid signed URL lifetime=%d actor=%+v", remaining, client.actor)
	}
}

func TestOSSDeletionFailureDoesNotFinishAndCanRetry(t *testing.T) {
	var fail atomic.Bool
	fail.Store(true)
	var calls atomic.Int64
	s, _ := fileOSS(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("unexpected method %s", r.Method)
		}
		calls.Add(1)
		if fail.Load() {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(503)
			_, _ = w.Write([]byte(`<Error><Code>ServiceUnavailable</Code><Message>temporary</Message></Error>`))
			return
		}
		w.WriteHeader(204)
	})
	client := &fileAPIClient{file: &pb.FileResource{ID: 4, ObjectKey: "go-zero-admin/test.png", Status: "deleting"}}
	s.AppletFileRPC = client
	l := NewDeleteFileLogic(fileAPIContext(), s)
	if _, err := l.DeleteFile(&types.FileIDRequest{Id: 4}); err == nil || client.finishCalls != 0 || client.file.Status != "deleting" {
		t.Fatalf("failed OSS deletion committed metadata: err=%v finish=%d", err, client.finishCalls)
	}
	fail.Store(false)
	if _, err := l.DeleteFile(&types.FileIDRequest{Id: 4}); err != nil || client.finishCalls != 1 || client.file.Status != "deleted" {
		t.Fatalf("retry failed: err=%v finish=%d", err, client.finishCalls)
	}
	before := calls.Load()
	if _, err := l.DeleteFile(&types.FileIDRequest{Id: 4}); err != nil || calls.Load() != before || client.finishCalls != 1 {
		t.Fatal("completed deletion not idempotent")
	}
}
