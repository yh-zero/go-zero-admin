package middleware

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"google.golang.org/grpc"
)

type recordingAudit struct {
	logs     []*pb.AuditLog
	canceled bool
}

func (r *recordingAudit) RecordAudit(ctx context.Context, req *pb.RecordAuditRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	r.canceled = ctx.Err() != nil
	r.logs = append(r.logs, req.Log)
	return &pb.NoDataResponse{}, nil
}

func TestAuditMiddlewareDetectsBusinessFailureAndPreservesBody(t *testing.T) {
	client := &recordingAudit{}
	body := `{"userId":3,"password":"secret","accessToken":"jwt","captcha":"otp"}`
	request := httptest.NewRequest(http.MethodPut, "/v1/sys/resetUserPassword", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-Forwarded-For", "spoofed")
	request.RemoteAddr = "192.0.2.10:1234"
	ctx, cancel := context.WithCancel(request.Context())
	cancel()
	request = request.WithContext(ctx)
	response := httptest.NewRecorder()
	NewAuditMiddleware(client).Handle(func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := io.ReadAll(r.Body)
		if string(bytes) != body {
			t.Fatal("middleware consumed request body")
		}
		audit.SetActor(r.Context(), audit.Actor{ID: 1, Name: "admin", AuthorityID: 1})
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code":900001,"message":"error"}`))
	})(response, request)
	if len(client.logs) != 1 {
		t.Fatal("missing audit")
	}
	log := client.logs[0]
	if log.Result != "failure" || log.StatusCode != 200 || log.ActorID != 1 || log.Object != "3" || log.IP != "192.0.2.10" || log.Params != `{"userid":3}` || client.canceled {
		t.Fatalf("unexpected audit: %+v canceled=%v", log, client.canceled)
	}
}

func TestAuditMiddlewareRecordsBothLoginOutcomesWithoutCredentials(t *testing.T) {
	client := &recordingAudit{}
	for _, success := range []bool{true, false} {
		request := httptest.NewRequest(http.MethodPost, "/v1/sys/login", strings.NewReader(`{"userName":"admin","password":"password-secret","captcha":"1234"}`))
		request.Header.Set("Content-Type", "application/json")
		NewAuditMiddleware(client).Handle(func(w http.ResponseWriter, r *http.Request) {
			if success {
				_, _ = w.Write([]byte(`{"code":200,"success":true,"result":{"accessToken":"token-secret","userInfo":{"id":4,"username":"admin","authorityId":1}}}`))
			} else {
				_, _ = w.Write([]byte(`{"code":900002,"message":"login failed"}`))
			}
		})(httptest.NewRecorder(), request)
	}
	if len(client.logs) != 2 || client.logs[0].ActorID != 4 || client.logs[0].Result != "success" || client.logs[1].ActorName != "admin" || client.logs[1].Result != "failure" {
		t.Fatalf("wrong login logs: %+v", client.logs)
	}
	encoded, _ := json.Marshal(client.logs)
	for _, secret := range []string{"token-secret", "password-secret", "1234"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("secret stored: %s", encoded)
		}
	}
}

func TestAuditMiddlewareSkipsReadsAndRestoresOversizedBody(t *testing.T) {
	client := &recordingAudit{}
	NewAuditMiddleware(client).Handle(func(w http.ResponseWriter, r *http.Request) {
		audit.SetActor(r.Context(), audit.Actor{ID: 1, Name: "admin", AuthorityID: 1})
		if audit.ActorFromContext(r.Context()).ID != 1 || audit.RequestFromContext(r.Context()).Path != "/v1/sys/audit/getAuditLogList" {
			t.Fatal("read request context not shared")
		}
		w.WriteHeader(200)
	})(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/v1/sys/audit/getAuditLogList", nil))
	if len(client.logs) != 0 {
		t.Fatal("read query audited")
	}
	body := `{"password":"` + strings.Repeat("x", 17000) + `"}`
	request := httptest.NewRequest(http.MethodPost, "/v1/sys/register", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	NewAuditMiddleware(client).Handle(func(w http.ResponseWriter, r *http.Request) {
		actual, _ := io.ReadAll(r.Body)
		if string(actual) != body {
			t.Fatal("oversized body corrupted")
		}
		w.WriteHeader(http.StatusBadRequest)
	})(httptest.NewRecorder(), request)
	if len(client.logs) != 1 || client.logs[0].Params != "{}" || client.logs[0].Result != "failure" {
		t.Fatal("oversized body not safely audited")
	}
}
