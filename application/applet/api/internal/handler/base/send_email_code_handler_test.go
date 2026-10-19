package base

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/rpc/client/mail"
	"go-zero-admin/application/applet/rpc/pb"
	"google.golang.org/grpc"
)

type mailIPClient struct {
	mail.Mail
	ip string
}

func (c *mailIPClient) SendEmailCode(_ context.Context, req *pb.SendEmailCodeRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	c.ip = req.ClientIP
	return &pb.NoDataResponse{}, nil
}

func TestLegacyEmailHandlerUsesTrustedPeerResolution(t *testing.T) {
	for _, tc := range []struct{ name, trusted, peer, forwarded, expected string }{
		{"direct spoof", "", "198.51.100.8:3141", "203.0.113.2", "198.51.100.8"},
		{"another port same IP", "", "198.51.100.8:9999", "", "198.51.100.8"},
		{"trusted proxy", "10.0.0.0/8", "10.0.0.2:3141", "203.0.113.2", "203.0.113.2"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resolver, err := appbridge.NewClientIPResolver(tc.trusted)
			if err != nil {
				t.Fatal(err)
			}
			client := &mailIPClient{}
			r := httptest.NewRequest("POST", "/v1/sys/sendEmailCode", strings.NewReader(`{"email":"test@example.com","clientIP":"192.0.2.11"}`))
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-Forwarded-For", tc.forwarded)
			r.RemoteAddr = tc.peer
			SendEmailCodeHandler(&svc.ServiceContext{ClientIPResolver: resolver, AppletMailRPC: client})(httptest.NewRecorder(), r)
			if client.ip != tc.expected {
				t.Fatalf("IP=%q want %q", client.ip, tc.expected)
			}
		})
	}
}
