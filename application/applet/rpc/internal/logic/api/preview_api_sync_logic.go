package apilogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/data/api/generated"

	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewApiSyncLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewPreviewApiSyncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewApiSyncLogic {
	return &PreviewApiSyncLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *PreviewApiSyncLogic) PreviewApiSync(_ *pb.NoDataResponse) (*pb.PreviewApiSyncResponse, error) {
	resources, err := readSwaggerResources(generated.Swagger)
	if err != nil {
		return nil, err
	}
	var records []model.SysApi
	if err := l.svcCtx.DB.WithContext(l.ctx).Order("id").Find(&records).Error; err != nil {
		return nil, err
	}
	return previewApiSync(resources, records)
}
