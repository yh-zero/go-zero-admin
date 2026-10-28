package api

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type PreviewApiSyncLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewPreviewApiSyncLogic(ctx context.Context, svcCtx *svc.ServiceContext) *PreviewApiSyncLogic {
	return &PreviewApiSyncLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *PreviewApiSyncLogic) PreviewApiSync() (*types.PreviewApiSyncResponse, error) {
	data, err := l.svcCtx.AppletAPIRPC.PreviewApiSync(l.ctx, &pb.NoDataResponse{})
	if err != nil {
		return nil, err
	}
	result := &types.PreviewApiSyncResponse{Added: []types.ApiSyncItem{}, Changed: []types.ApiSyncItem{}, Obsolete: []types.ApiSyncItem{}}
	if err := copier.Copy(result, data); err != nil {
		return nil, err
	}
	// Protobuf may decode an empty repeated field as nil. Keep the JSON lists
	// consistently iterable even when a synchronization category is empty.
	if result.Added == nil {
		result.Added = []types.ApiSyncItem{}
	}
	if result.Changed == nil {
		result.Changed = []types.ApiSyncItem{}
	}
	if result.Obsolete == nil {
		result.Obsolete = []types.ApiSyncItem{}
	}
	return result, nil
}
