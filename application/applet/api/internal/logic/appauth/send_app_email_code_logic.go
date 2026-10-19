// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"context"
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendAppEmailCodeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 发送 App 邮箱注册验证码
func NewSendAppEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendAppEmailCodeLogic {
	return &SendAppEmailCodeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *SendAppEmailCodeLogic) SendAppEmailCode(req *types.AppSendEmailCodeRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppAccountRPC.SendAppEmailCode(l.ctx, &pb.AppSendEmailCodeRequest{Account: req.Account, IP: appbridge.Request(l.ctx).IP})
	if err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "验证码已发送，5分钟内有效"}, nil
}
