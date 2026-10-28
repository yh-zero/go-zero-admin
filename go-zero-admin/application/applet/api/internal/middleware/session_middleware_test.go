package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"go-zero-admin/application/applet/rpc/client/user"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"google.golang.org/grpc"
	"net/http"
	"net/http/httptest"
	"testing"
)

type sessionStub struct {
	user.User
	valid bool
	err   error
	got   *pb.SessionRequest
}

func (s *sessionStub) CheckSession(_ context.Context, req *pb.SessionRequest, _ ...grpc.CallOption) (*pb.CheckSessionResponse, error) {
	s.got = req
	return &pb.CheckSessionResponse{Valid: s.valid}, s.err
}
func TestSessionMiddleware(t *testing.T) {
	for _, test := range []struct {
		name    string
		valid   bool
		err     error
		status  int
		allowed bool
	}{
		{name: "active", valid: true, status: 200, allowed: true},
		{name: "revoked", status: 401},
		{name: "rpc unavailable", err: errors.New("unavailable"), status: 503},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &sessionStub{valid: test.valid, err: test.err}
			called := false
			handler := NewSessionMiddleware(client).Handle(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(200) })
			request := httptest.NewRequest("GET", "/v1/sys/me", nil)
			request = request.WithContext(context.WithValue(request.Context(), ctxJwt.CtxKeyJwtData, map[string]any{"ID": json.Number("12"), "AuthorityId": json.Number("801"), "SessionVersion": json.Number("3")}))
			recorder := httptest.NewRecorder()
			handler(recorder, request)
			if recorder.Code != test.status || called != test.allowed {
				t.Fatalf("status=%d next=%v", recorder.Code, called)
			}
			if client.got.UserID != 12 || client.got.AuthorityId != 801 || client.got.SessionVersion != 3 {
				t.Fatalf("lost trusted claims: %+v", client.got)
			}
		})
	}
}
