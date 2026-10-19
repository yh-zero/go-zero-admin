package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type SendAppEmailCodeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSendAppEmailCodeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendAppEmailCodeLogic {
	return &SendAppEmailCodeLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *SendAppEmailCodeLogic) SendAppEmailCode(in *pb.AppSendEmailCodeRequest) (*pb.NoDataResponse, error) {
	return sendEmailCode(l.ctx, l.svcCtx, in)
}
