package userlogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetUserResourcePreviewLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetUserResourcePreviewLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserResourcePreviewLogic {
	return &GetUserResourcePreviewLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetUserResourcePreviewLogic) GetUserResourcePreview(in *pb.UserResourcePreviewRequest) (*pb.UserResourcePreviewResponse, error) {
	return previewUserResources(l.ctx, l.svcCtx.DB.DB, in, l.svcCtx.Config.DB.DataSource, l.svcCtx.Config.UserResources.AIDataSource)
}
