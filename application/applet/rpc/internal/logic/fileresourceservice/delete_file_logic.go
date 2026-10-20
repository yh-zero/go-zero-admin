package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// DeleteFile composes BeginDeleteFile -> object storage delete -> FinishDeleteFile so the
// row lock is never held across the network call, and every phase keeps its
// existing audit events. The composition is idempotent: Begin short-circuits
// on "deleted"/"deleting" rows, S3 DELETE of a missing object succeeds, and
// Finish short-circuits on "deleted". A crash between phases leaves the row
// "deleting"; the next call resumes from exactly that point.
func (l *DeleteFileLogic) DeleteFile(in *pb.FileIDRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "文件ID无效")
	}
	storage, err := configuredStorage(l.svcCtx)
	if err != nil {
		return nil, err
	}
	file, err := NewBeginDeleteFileLogic(l.ctx, l.svcCtx).BeginDeleteFile(in)
	if err != nil {
		return nil, err
	}
	if file.Status == "deleted" {
		return &pb.NoDataResponse{}, nil
	}
	if err := storage.Delete(l.ctx, file.ObjectKey, file.Visibility); err != nil {
		l.Errorf("object storage file deletion failed id=%d", file.ID)
		// The row stays "deleting"; the caller can retry the same RPC.
		return nil, xerr.NewErrCodeMsg(300002, "存储文件删除失败，资源已标记待删除，可再次重试")
	}
	return NewFinishDeleteFileLogic(l.ctx, l.svcCtx).FinishDeleteFile(in)
}
