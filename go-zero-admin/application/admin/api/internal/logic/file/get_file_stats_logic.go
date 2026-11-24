package file

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileStatsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFileStatsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileStatsLogic {
	return &GetFileStatsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetFileStatsLogic) GetFileStats(req *types.GetFileStatsReq) (*types.GetFileStatsResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	userId, _ := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	if userId <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	resp, err := l.svcCtx.FileRpc.GetFileStats(l.ctx, &filepb.GetFileStatsRequest{Actor: &filepb.Actor{UserID: userId}})
	if err != nil {
		return nil, err
	}
	return &types.GetFileStatsResp{
		TotalFiles: resp.TotalFiles, TotalBytes: resp.TotalBytes,
		PublicFiles: resp.PublicFiles, PublicBytes: resp.PublicBytes,
		PrivateFiles: resp.PrivateFiles, PrivateBytes: resp.PrivateBytes,
		CleanupPendingFiles: resp.CleanupPendingFiles, CleanupPendingBytes: resp.CleanupPendingBytes,
	}, nil
}
