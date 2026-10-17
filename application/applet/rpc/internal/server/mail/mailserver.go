// Code scaffolded by goctl. Safe to edit.

package server

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/logic/mail"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type MailServer struct {
	svcCtx *svc.ServiceContext
	pb.UnimplementedMailServer
}

func NewMailServer(svcCtx *svc.ServiceContext) *MailServer {
	return &MailServer{svcCtx: svcCtx}
}

func (s *MailServer) SendEmailCode(ctx context.Context, in *pb.SendEmailCodeRequest) (*pb.NoDataResponse, error) {
	l := maillogic.NewSendEmailCodeLogic(ctx, s.svcCtx)
	return l.SendEmailCode(in)
}

func (s *MailServer) VerifyEmailCode(ctx context.Context, in *pb.VerifyEmailCodeRequest) (*pb.VerifyEmailCodeResponse, error) {
	l := maillogic.NewVerifyEmailCodeLogic(ctx, s.svcCtx)
	return l.VerifyEmailCode(in)
}
