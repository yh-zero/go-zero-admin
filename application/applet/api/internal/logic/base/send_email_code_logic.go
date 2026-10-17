package base

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendEmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendEmailCodeLogic {
	return &SendEmailCodeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

// SendEmailCode is a thin pass-through to the RPC tier. The RPC owns the SMTP
// credentials, Redis cooldown / IP / per-email rate limits, and the atomic
// code generation. The API only forwards the request (with the originating IP
// for IP-based rate limiting).
func (l *SendEmailCodeLogic) SendEmailCode(req *types.SendEmailCodeRequest) (*types.MessageResponse, error) {
	if l.svcCtx.AppletMailRPC == nil {
		return nil, xerr.NewErrCodeMsg(900000, "邮件服务未配置")
	}
	if _, err := l.svcCtx.AppletMailRPC.SendEmailCode(l.ctx, &pb.SendEmailCodeRequest{
		Email:    req.Email,
		IsForce:  req.IsForce,
		Scene:    req.Scene,
		ClientIP: req.ClientIP,
	}); err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "验证码发送成功"}, nil
}
