package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type RegisterAppAccountLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterAppAccountLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterAppAccountLogic {
	return &RegisterAppAccountLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterAppAccountLogic) RegisterAppAccount(in *pb.AppRegisterRequest) (*pb.AppUserInfo, error) {
	return register(l.ctx, l.svcCtx, in)
}
