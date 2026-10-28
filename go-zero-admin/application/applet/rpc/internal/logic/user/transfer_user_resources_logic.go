package userlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type TransferUserResourcesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewTransferUserResourcesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *TransferUserResourcesLogic {
	return &TransferUserResourcesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *TransferUserResourcesLogic) TransferUserResources(in *pb.TransferUserResourcesRequest) (*pb.TransferUserResourcesResponse, error) {
	return transferUserResources(l.ctx, l.svcCtx.DB.DB, in, l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.UserResources.AIDataSource)
}
