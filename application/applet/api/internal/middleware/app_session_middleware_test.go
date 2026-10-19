package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/client/appaccountservice"
	"go-zero-admin/application/applet/rpc/pb"

	"google.golang.org/grpc"
)

type appSessionStub struct {
	appaccountservice.AppAccountService
	metadata appbridge.Metadata
	token    string
	err      error
}

func (s *appSessionStub) CheckAppSession(ctx context.Context, req *pb.AppAccessRequest, _ ...grpc.CallOption) (*pb.AppUserInfo, error) {
	s.metadata = appbridge.Request(ctx)
	s.token = req.AccessToken
	return &pb.AppUserInfo{}, s.err
}

func TestAppMiddlewaresShareTrustedClientIP(t *testing.T) {
	resolver, err := appbridge.NewClientIPResolver("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	client := &appSessionStub{}
	for name, wrap := range map[string]func(http.HandlerFunc) http.HandlerFunc{
		"request": AppRequestWithResolver(resolver),
		"session": AppSessionWithResolver(client, resolver),
	} {
		t.Run(name, func(t *testing.T) {
			for _, ip := range []string{"203.0.113.8", "203.0.113.9"} {
				var got appbridge.Metadata
				handler := wrap(func(w http.ResponseWriter, r *http.Request) { got = appbridge.Request(r.Context()) })
				r := httptest.NewRequest("GET", "/v1/app/me", nil)
				r.RemoteAddr = "10.0.0.2:1234"
				r.Header.Set("X-Forwarded-For", ip)
				r.Header.Set("Authorization", "Bearer app-token")
				r.Header.Set("User-Agent", "app-test")
				handler(httptest.NewRecorder(), r)
				if got.IP != ip || got.Token != "app-token" || got.UserAgent != "app-test" {
					t.Fatalf("handler metadata=%+v, want client IP=%q and original token/user agent", got, ip)
				}
				if name == "session" && (client.metadata != got || client.token != "app-token") {
					t.Fatalf("session check lost request identity: metadata=%+v token=%q", client.metadata, client.token)
				}
			}
		})
	}
}

func TestAppSessionStillRejectsInvalidSession(t *testing.T) {
	client := &appSessionStub{err: errors.New("invalid app session")}
	called := false
	handler := AppSession(client)(func(http.ResponseWriter, *http.Request) { called = true })
	r := httptest.NewRequest("GET", "/v1/app/me", nil)
	r.RemoteAddr = "198.51.100.9:1234"
	r.Header.Set("X-Forwarded-For", "203.0.113.8")
	r.Header.Set("Authorization", "Bearer app-token")
	handler(httptest.NewRecorder(), r)
	if called {
		t.Fatal("invalid app session reached the protected handler")
	}
	if client.metadata.IP != "198.51.100.9" || client.token != "app-token" {
		t.Fatalf("default session middleware lost peer identity: %+v token=%q", client.metadata, client.token)
	}
}
