package appaccountservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetAppAuthOptionsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAppAuthOptionsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAppAuthOptionsLogic {
	return &GetAppAuthOptionsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetAppAuthOptionsLogic) GetAppAuthOptions(in *pb.NoDataResponse) (*pb.AppAuthOptions, error) {
	return authOptions(l.svcCtx)
}
