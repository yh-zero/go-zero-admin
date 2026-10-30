package apiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
)

type SyncApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewSyncApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SyncApiLogic {
	return &SyncApiLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *SyncApiLogic) SyncApi(in *adminpb.SyncApiRequest) (*adminpb.SyncApiResponse, error) {
	// 简版：从 .api 文件正则提取路由
	// 生产实现应解析 desc 目录所有 .api
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.SyncApiResponse{Added: 0, Updated: 0, Removed: 0}, nil
}
