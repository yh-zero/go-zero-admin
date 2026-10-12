// Code scaffolded by goctl. Safe to edit.

package server

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/captcha"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type CaptchaServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedCaptchaServer
}

func NewCaptchaServer(svcCtx *svc.ServiceContext) *CaptchaServer {
	return &CaptchaServer{svcCtx: svcCtx}
}

func (s *CaptchaServer) GenerateImage(ctx context.Context, in *pb.NoDataResponse) (*pb.GenerateImageResponse, error) {
	l := captchalogic.NewGenerateImageLogic(ctx, s.svcCtx)
	return l.GenerateImage(in)
}

func (s *CaptchaServer) VerifyCaptcha(ctx context.Context, in *pb.VerifyCaptchaRequest) (*pb.VerifyCaptchaResponse, error) {
	l := captchalogic.NewVerifyCaptchaLogic(ctx, s.svcCtx)
	return l.VerifyCaptcha(in)
}
