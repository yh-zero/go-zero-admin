package appauthservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	apppb "go-zero-admin/application/app/rpc/apppb"
	"go-zero-admin/application/app/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type SendEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SendEmailCodeLogic) SendEmailCode(in *apppb.AppSendEmailCodeRequest) (*apppb.NoDataResponse, error) {
	// 简版：SMTP 未配置，直接返回错误
	return nil, xerr.NewErrCodeMsg(400, "邮箱验证码未启用")
}
