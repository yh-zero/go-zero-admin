package appauthservicelogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
)

type GetAuthOptionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthOptionsLogic(ctx context.Context, s *svc.ServiceContext) *GetAuthOptionsLogic {
	return &GetAuthOptionsLogic{ctx, s, logx.WithContext(ctx)}
}
func (l *GetAuthOptionsLogic) GetAuthOptions(in *apppb.AppAuthOptionsRequest) (*apppb.AppAuthOptionsResponse, error) {
	return &apppb.AppAuthOptionsResponse{EmailVerificationRequired: l.svcCtx.Config.EmailVerificationRequired, EmailCodeEnabled: smtpEnabled(l.svcCtx), PhoneEnabled: false}, nil
}
