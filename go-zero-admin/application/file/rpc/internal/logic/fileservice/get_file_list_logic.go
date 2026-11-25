package fileservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/model"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetFileListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileListLogic {
	return &GetFileListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetFileListLogic) GetFileList(in *filepb.GetFileListRequest) (*filepb.GetFileListResponse, error) {
	var viewer int64
	if in.GetIncludeCleanupPending() || in.GetCleanupPendingOnly() {
		viewer = in.GetActor().GetUserID()
	}
	rows, total, err := l.svcCtx.SysFileModel.ListFilePage(l.ctx, model.FileListFilter{
		Visibility: in.Visibility, UploaderId: in.UploaderID, ViewerId: viewer,
		CleanupPendingOnly: in.CleanupPendingOnly, Name: in.Name, Mime: in.Mime, Ext: in.Ext,
		MinSize: in.MinSize, MaxSize: in.MaxSize, StartTime: in.StartTime, EndTime: in.EndTime,
	}, in.GetPage().GetPage(), in.GetPage().GetPageSize())
	if errors.Is(err, model.ErrInvalidFileFilter) {
		return nil, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件筛选大小、时间或清理状态不合法")
	}
	if err != nil {
		l.Errorf("query files error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*filepb.FileInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &filepb.FileInfo{
			Id: r.Id, Bucket: r.Bucket, ObjectKey: r.ObjectKey, OriginalName: r.OriginalName,
			Ext: r.Ext, Mime: r.Mime, Size: r.Size, Visibility: r.Visibility,
			RefCount: r.RefCount, UploaderID: r.UploaderId, UploaderName: r.UploaderName, Sha256: r.SHA256,
			CreatedAt: r.CreatedAt, CleanupPending: r.CleanupPending,
		})
	}
	return &filepb.GetFileListResponse{List: list, Page: &filepb.PageResponse{Total: total}}, nil
}
