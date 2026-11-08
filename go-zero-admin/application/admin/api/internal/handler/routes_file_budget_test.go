package handler

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
	"go-zero-admin/application/admin/api/internal/config"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/file/rpc/client/fileservice"
	"go-zero-admin/application/file/rpc/filepb"
	"google.golang.org/grpc"
)

const uploadBudgetBody = `{"name":"budget.txt","content_base64":"aGVsbG8=","scene":"doc","visibility":"private"}`

func TestFileUploadBudgetCoversStorageWithoutExtendingList(t *testing.T) {
	installRouteResponseHandlers(t)
	for _, mode := range []string{"host", "compose"} {
		t.Run(mode, func(t *testing.T) {
			apiConfig, rpcConfig, storageSeconds := loadFileBudgetConfig(t, mode)
			if rpcConfig.Timeout <= storageSeconds*1000 {
				t.Errorf("文件RPC服务端预算 %dms 必须覆盖存储 %ds", rpcConfig.Timeout, storageSeconds)
			}
			if apiConfig.FileRpc.Timeout <= rpcConfig.Timeout {
				t.Errorf("文件RPC客户端预算 %dms 必须覆盖服务端 %dms", apiConfig.FileRpc.Timeout, rpcConfig.Timeout)
			}
			client := &budgetFileClient{check: func(ctx context.Context, upload bool) {
				deadline, ok := ctx.Deadline()
				if !ok {
					t.Error("文件路由缺少请求截止时间")
					return
				}
				remaining := time.Until(deadline)
				if upload && remaining <= time.Duration(apiConfig.FileRpc.Timeout)*time.Millisecond {
					t.Errorf("上传HTTP剩余预算 %v 未覆盖文件RPC %dms", remaining, apiConfig.FileRpc.Timeout)
				}
				if !upload && (remaining > 3*time.Second || remaining < time.Second) {
					t.Errorf("文件列表应保留短请求预算，实际 %v", remaining)
				}
			}}
			router := compiledFileBudgetRouter(t, apiConfig, client)
			for _, endpoint := range []struct{ method, path, body string }{
				{http.MethodPost, "/v1/sys/file/upload", uploadBudgetBody},
				{http.MethodGet, "/v1/sys/file/list", ""},
			} {
				r := httptest.NewRequest(endpoint.method, endpoint.path, strings.NewReader(endpoint.body))
				r.Header.Set("Content-Type", "application/json")
				recorder := httptest.NewRecorder()
				router.ServeHTTP(recorder, r)
				assertBudgetSuccess(t, recorder)
			}
		})
	}
}

func TestFileUploadConfiguredClientAllowsSlowRPCAndPropagatesCancellation(t *testing.T) {
	installRouteResponseHandlers(t)
	apiConfig, _, _ := loadFileBudgetConfig(t, "compose")
	for _, test := range []struct {
		name   string
		delay  time.Duration
		cancel bool
	}{
		{name: "超过旧2秒预算仍成功", delay: 2200 * time.Millisecond},
		{name: "调用方取消传递到慢RPC", delay: time.Minute, cancel: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			server := grpc.NewServer()
			stub := &slowFileBudgetServer{delay: test.delay, canceled: make(chan struct{})}
			filepb.RegisterFileServiceServer(server, stub)
			go func() { _ = server.Serve(listener) }()
			t.Cleanup(func() { server.Stop(); _ = listener.Close() })
			t.Logf("慢文件RPC桩 %s，测试结束精确关闭", listener.Addr())
			clientConfig := apiConfig.FileRpc
			clientConfig.Endpoints = []string{listener.Addr().String()}
			clientConfig.Middlewares.Duration = false
			client, err := zrpc.NewClient(clientConfig)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = client.Conn().Close() })
			router := compiledFileBudgetRouter(t, apiConfig, fileservice.NewFileService(client))
			r := httptest.NewRequest(http.MethodPost, "/v1/sys/file/upload", strings.NewReader(uploadBudgetBody))
			r.Header.Set("Content-Type", "application/json")
			if test.cancel {
				ctx, cancel := context.WithTimeout(r.Context(), 100*time.Millisecond)
				defer cancel()
				r = r.WithContext(ctx)
			}
			recorder := httptest.NewRecorder()
			router.ServeHTTP(recorder, r)
			if !test.cancel {
				assertBudgetSuccess(t, recorder)
				return
			}
			select {
			case <-stub.canceled:
			case <-time.After(2 * time.Second):
				t.Fatal("调用方取消没有传递到RPC，后台调用仍占用资源")
			}
			var body struct {
				Code int `json:"code"`
			}
			if json.Unmarshal(recorder.Body.Bytes(), &body) == nil && body.Code == 200 {
				t.Fatal("取消的上传不能报告成功")
			}
		})
	}
}

func loadFileBudgetConfig(t *testing.T, mode string) (config.Config, zrpc.RpcServerConf, int64) {
	t.Helper()
	root := filepath.Join("..", "..", "..", "..", "..")
	apiPath, filePath := "application/admin/api/etc/admin.yaml", "application/file/rpc/etc/file.yaml"
	if mode == "compose" {
		apiPath, filePath = "docker/etc/admin-api.yaml", "docker/etc/file-rpc.yaml"
	}
	var apiConfig config.Config
	if err := conf.Load(filepath.Join(root, apiPath), &apiConfig, conf.UseEnv()); err != nil {
		t.Fatal(err)
	}
	var fileConfig struct {
		zrpc.RpcServerConf
		Storage struct{ RequestTimeoutSeconds int64 }
	}
	if err := conf.Load(filepath.Join(root, filePath), &fileConfig, conf.UseEnv()); err != nil {
		t.Fatal(err)
	}
	return apiConfig, fileConfig.RpcServerConf, fileConfig.Storage.RequestTimeoutSeconds
}

// 使用框架公开启动选项取得已完成路由绑定的 handler，并在监听前停止启动。
// 随后仅用 httptest 调用真实中间件链，不启动项目服务或访问数据库。
func compiledFileBudgetRouter(t *testing.T, c config.Config, client fileservice.FileService) http.Handler {
	t.Helper()
	c.Name, c.Mode, c.Log.Mode = "file-upload-budget-test", "test", "console"
	c.DevServer.Enabled, c.Middlewares.Log = false, false
	server, err := rest.NewServer(c.RestConf)
	if err != nil {
		t.Fatal(err)
	}
	RegisterHandlers(server, &svc.ServiceContext{Config: c, FileRpc: client, Validator: svc.NewValidator()})
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), middleware.CtxKeyUserID, int64(7))
			next(w, r.WithContext(ctx))
		}
	})
	type stopBeforeListen struct{}
	var router http.Handler
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				if _, ok := recovered.(stopBeforeListen); !ok {
					panic(recovered)
				}
			}
		}()
		server.StartWithOpts(func(httpServer *http.Server) {
			router = httpServer.Handler
			panic(stopBeforeListen{})
		})
	}()
	if router == nil {
		t.Fatal("未取得实际注册路由")
	}
	return router
}

func assertBudgetSuccess(t *testing.T, recorder *httptest.ResponseRecorder) {
	t.Helper()
	var body struct {
		Code int `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || recorder.Code != http.StatusOK || body.Code != 200 {
		t.Fatalf("文件请求没有成功：HTTP=%d，body=%s，err=%v", recorder.Code, recorder.Body.String(), err)
	}
}

type budgetFileClient struct {
	fileservice.FileService
	check func(context.Context, bool)
}

func (c *budgetFileClient) UploadFile(ctx context.Context, _ *filepb.UploadFileRequest, _ ...grpc.CallOption) (*filepb.UploadFileResponse, error) {
	c.check(ctx, true)
	return &filepb.UploadFileResponse{File: &filepb.FileInfo{Id: 1, OriginalName: "budget.txt"}}, nil
}

func (c *budgetFileClient) GetFileList(ctx context.Context, _ *filepb.GetFileListRequest, _ ...grpc.CallOption) (*filepb.GetFileListResponse, error) {
	c.check(ctx, false)
	return &filepb.GetFileListResponse{Page: &filepb.PageResponse{}}, nil
}

type slowFileBudgetServer struct {
	filepb.UnimplementedFileServiceServer
	delay    time.Duration
	canceled chan struct{}
}

func (s *slowFileBudgetServer) UploadFile(ctx context.Context, _ *filepb.UploadFileRequest) (*filepb.UploadFileResponse, error) {
	select {
	case <-time.After(s.delay):
		return &filepb.UploadFileResponse{File: &filepb.FileInfo{Id: 1, OriginalName: "budget.txt"}}, nil
	case <-ctx.Done():
		close(s.canceled)
		return nil, ctx.Err()
	}
}
