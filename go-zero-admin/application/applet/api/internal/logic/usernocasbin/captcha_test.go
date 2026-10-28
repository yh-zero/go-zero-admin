package usernocasbin

import (
	"context"
	"errors"
	"testing"

	"go-zero-admin/application/applet/api/internal/svc"
	userclient "go-zero-admin/application/applet/rpc/client/user"

	"go-zero-admin/application/applet/api/internal/types"
	captchaRPC "go-zero-admin/application/applet/rpc/client/captcha"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"google.golang.org/grpc"
)

type stubCaptcha struct {
	captchaRPC.Captcha
	request             *pb.VerifyCaptchaRequest
	resp                *pb.VerifyCaptchaResponse
	err                 error
	generateImageCalled bool
}

func (s *stubCaptcha) VerifyCaptcha(_ context.Context, req *pb.VerifyCaptchaRequest, _ ...grpc.CallOption) (*pb.VerifyCaptchaResponse, error) {
	s.request = req
	return s.resp, s.err
}

func (s *stubCaptcha) GenerateImage(_ context.Context, _ *pb.NoDataResponse, _ ...grpc.CallOption) (*pb.GenerateImageResponse, error) {
	s.generateImageCalled = true
	return &pb.GenerateImageResponse{Id: "captcha-123", Image: "data:image/png;base64,AAAA"}, nil
}

func TestLoginPassesCaptchaIdAndAnswerToRPC(t *testing.T) {
	cstub := &stubCaptcha{resp: &pb.VerifyCaptchaResponse{Matched: true}}
	ustub := &stubUser{}
	s := &svc.ServiceContext{AppletCaptchaRPC: cstub, AppletUserRPC: ustub}
	_, err := NewLoginLogic(context.Background(), s).Login(&types.LoginRequest{UserName: "u", Password: "p", CaptchaId: "id-1", Captcha: "123456"})
	if cstub.request == nil || cstub.request.Id != "id-1" || cstub.request.Answer != "123456" {
		t.Fatalf("captcha not forwarded: %+v", cstub.request)
	}
	if !ustub.getUserInfoCalled {
		t.Fatal("login must proceed past captcha to user lookup")
	}
	if !errors.Is(err, errStopAfterCaptcha) {
		t.Fatalf("expected downstream error to surface, got %v", err)
	}
}

func TestLoginRejectsOnCaptchaMismatch(t *testing.T) {
	stub := &stubCaptcha{resp: &pb.VerifyCaptchaResponse{Matched: false}}
	s := &svc.ServiceContext{AppletCaptchaRPC: stub}
	_, err := NewLoginLogic(context.Background(), s).Login(&types.LoginRequest{UserName: "u", Password: "p", CaptchaId: "id-1", Captcha: "123456"})
	var be *xerr.CodeError
	if !errors.As(err, &be) || be.GetErrCode() != xerr.CAPTCHA_ERROR {
		t.Fatalf("expected CAPTCHA_ERROR, got %v", err)
	}
}

func TestRandomImagePassesThrough(t *testing.T) {
	stub := &stubCaptcha{}
	s := &svc.ServiceContext{AppletCaptchaRPC: stub}
	resp, err := NewRandomImageLogic(context.Background(), s).RandomImage(&types.RandomImageRequest{})
	if err != nil {
		t.Fatal(err)
	}
	if !stub.generateImageCalled {
		t.Fatal("GenerateImage RPC not called")
	}
	if resp.CaptchaId != "captcha-123" || resp.CaptchaImg == "" {
		t.Fatalf("response not mapped: %+v", resp)
	}
}

var errStopAfterCaptcha = errors.New("stub: stop after captcha")

type stubUser struct {
	userclient.User
	getUserInfoCalled bool
}

func (s *stubUser) GetUserInfo(_ context.Context, _ *pb.GetUserInfoRequest, _ ...grpc.CallOption) (*pb.GetUserInfoResponse, error) {
	s.getUserInfoCalled = true
	return nil, errStopAfterCaptcha
}
