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

func NewGetAuthOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthOptionsLogic {
	return &GetAuthOptionsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetAuthOptionsLogic) GetAuthOptions(in *apppb.AppAuthOptionsRequest) (*apppb.AppAuthOptionsResponse, error) {
	// 简版：返回固定配置；完整版读 SMTP/开关配置
	return &apppb.AppAuthOptionsResponse{
		EmailVerificationRequired: false,
		EmailCodeEnabled:          false,
		PhoneEnabled:              true,
	}, nil
}
