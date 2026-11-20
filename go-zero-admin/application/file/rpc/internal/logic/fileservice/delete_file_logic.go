package fileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteFileLogic) DeleteFile(in *filepb.DeleteFileRequest) (*filepb.NoDataResponse, error) {
	if in.GetActor().GetUserID() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	if in.FileID <= 0 {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	if l.svcCtx.Store == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "对象存储未配置")
	}
	row, err := l.svcCtx.SysFileModel.MarkDeletedIfUnused(l.ctx, in.FileID, in.Actor.UserID)
	if err != nil {
		return nil, fileModelError(err)
	}
	if row.ObjectCleanedAt.Valid {
		return &filepb.NoDataResponse{}, nil
	}
	if err := l.svcCtx.Store.DeleteFromBucket(l.ctx, row.Bucket, row.ObjectKey); err != nil {
		l.Error("文件对象删除失败，元数据已软删除，可重试清理")
		return nil, xerr.NewErrCodeMsg(xerr.SERVER_COMMON_ERROR, "存储清理失败，请重试删除")
	}
	if err := l.svcCtx.SysFileModel.ConfirmObjectCleanup(l.ctx, in.FileID); err != nil {
		l.Error("写入对象清理回执失败")
		return nil, xerr.NewErrCodeMsg(xerr.DB_ERROR, "文件清理结果尚未确认，请重试删除")
	}
	return &filepb.NoDataResponse{}, nil
}
