package fileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
)

type GetFileStatsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileStatsLogic {
	return &GetFileStatsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetFileStatsLogic) GetFileStats(in *filepb.GetFileStatsRequest) (*filepb.GetFileStatsResponse, error) {
	stats, err := l.svcCtx.SysFileModel.GetFileStats(l.ctx, in.GetActor().GetUserID())
	if err != nil {
		return nil, fileModelError(err)
	}
	return &filepb.GetFileStatsResponse{
		TotalFiles: stats.TotalFiles, TotalBytes: stats.TotalBytes,
		PublicFiles: stats.PublicFiles, PublicBytes: stats.PublicBytes,
		PrivateFiles: stats.PrivateFiles, PrivateBytes: stats.PrivateBytes,
		CleanupPendingFiles: stats.CleanupPendingFiles, CleanupPendingBytes: stats.CleanupPendingBytes,
	}, nil
}
