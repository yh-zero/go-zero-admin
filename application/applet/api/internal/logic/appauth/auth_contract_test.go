package appauth

import (
	"context"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/client/appaccountservice"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc"
)

type authClient struct {
	appaccountservice.AppAccountService
	registration *pb.AppRegisterRequest
	email        *pb.AppSendEmailCodeRequest
	options      *pb.AppAuthOptions
	err          error
}

func (c *authClient) RegisterAppAccount(_ context.Context, req *pb.AppRegisterRequest, _ ...grpc.CallOption) (*pb.AppUserInfo, error) {
	c.registration = req
	return &pb.AppUserInfo{ID: 17, Account: req.Account, IdentityKind: "email", IdentityVerified: true, Status: "pending"}, c.err
}
func (c *authClient) SendAppEmailCode(_ context.Context, req *pb.AppSendEmailCodeRequest, _ ...grpc.CallOption) (*pb.NoDataResponse, error) {
	c.email = req
	return &pb.NoDataResponse{}, c.err
}
func (c *authClient) GetAppAuthOptions(_ context.Context, _ *pb.NoDataResponse, _ ...grpc.CallOption) (*pb.AppAuthOptions, error) {
	return c.options, c.err
}

func TestRegistrationPreservesVerificationAndPendingStatus(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/app/auth/register", strings.NewReader(`{"account":"member@example.test","password":"a-test-password","nickname":"member","emailCode":"456789","ip":"spoofed"}`))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-Forwarded-For", "192.0.2.200")
	r.RemoteAddr = "198.51.100.9:3456"
	r = appbridge.WithRequest(r)
	var input types.AppRegisterRequest
	if err := httpx.Parse(r, &input); err != nil {
		t.Fatal(err)
	}
	client := &authClient{}
	result, err := NewRegisterAppAccountLogic(r.Context(), &svc.ServiceContext{AppAccountRPC: client}).RegisterAppAccount(&input)
	if err != nil {
		t.Fatal(err)
	}
	got := client.registration
	if got.EmailCode != "456789" || got.Account != "member@example.test" || got.Password != "a-test-password" || got.Nickname != "member" || got.IP != "198.51.100.9" {
		t.Fatalf("registration lost verification fields or used spoofed IP: %+v", got)
	}
	if result.ID != 17 || !result.IdentityVerified || result.Status != "pending" || result.IdentityKind != "email" {
		t.Fatalf("registration incorrectly converted verified email into account approval: %+v", result)
	}
}

func TestEmailCodeUsesResolvedProxyClientIP(t *testing.T) {
	resolver, err := appbridge.NewClientIPResolver("10.20.0.0/16")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/v1/app/auth/email-code", nil)
	r.RemoteAddr = "10.20.0.3:3456"
	r.Header.Set("X-Forwarded-For", "203.0.113.9")
	r = resolver.WithRequest(r)
	client := &authClient{}
	response, err := NewSendAppEmailCodeLogic(r.Context(), &svc.ServiceContext{AppAccountRPC: client}).SendAppEmailCode(&types.AppSendEmailCodeRequest{Account: "member@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	if response.Message == "" || client.email.Account != "member@example.test" || client.email.IP != "203.0.113.9" {
		t.Fatalf("email code must use request metadata for rate limits: %+v", client.email)
	}
}

func TestAuthOptionsPreserveIndependentFlags(t *testing.T) {
	for _, required := range []bool{false, true} {
		for _, enabled := range []bool{false, true} {
			client := &authClient{options: &pb.AppAuthOptions{EmailVerificationRequired: required, EmailCodeEnabled: enabled}}
			result, err := NewGetAppAuthOptionsLogic(context.Background(), &svc.ServiceContext{AppAccountRPC: client}).GetAppAuthOptions()
			if err != nil {
				t.Fatal(err)
			}
			if result.EmailVerificationRequired != required || result.EmailCodeEnabled != enabled {
				t.Fatalf("options changed: %+v", result)
			}
		}
	}
}

func TestEmailAuthRPCFailuresRemainFailures(t *testing.T) {
	want := errors.New("mail service unavailable")
	service := &svc.ServiceContext{AppAccountRPC: &authClient{err: want}}
	t.Run("register", func(t *testing.T) {
		result, err := NewRegisterAppAccountLogic(context.Background(), service).RegisterAppAccount(&types.AppRegisterRequest{})
		if result != nil || !errors.Is(err, want) {
			t.Fatalf("error changed: result=%+v error=%v", result, err)
		}
	})
	t.Run("send", func(t *testing.T) {
		result, err := NewSendAppEmailCodeLogic(context.Background(), service).SendAppEmailCode(&types.AppSendEmailCodeRequest{})
		if result != nil || !errors.Is(err, want) {
			t.Fatalf("error changed: result=%+v error=%v", result, err)
		}
	})
	t.Run("options", func(t *testing.T) {
		result, err := NewGetAppAuthOptionsLogic(context.Background(), service).GetAppAuthOptions()
		if result != nil || !errors.Is(err, want) {
			t.Fatalf("error changed: result=%+v error=%v", result, err)
		}
	})
}
