package fileservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
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
	where := "deleted_at IS NULL"
	var args []interface{}
	if in.Visibility != "" {
		where += " AND visibility=?"
		args = append(args, in.Visibility)
	}
	if in.UploaderID > 0 {
		where += " AND uploader_id=?"
		args = append(args, in.UploaderID)
	}

	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM sys_file WHERE "+where, args...)
	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	type fileRow struct {
		Id           int64  `db:"id"`
		Bucket       string `db:"bucket"`
		ObjectKey    string `db:"object_key"`
		OriginalName string `db:"original_name"`
		Ext          string `db:"ext"`
		Mime         string `db:"mime"`
		Size         int64  `db:"size"`
		Visibility   string `db:"visibility"`
		RefCount     int32  `db:"ref_count"`
		UploaderId   int64  `db:"uploader_id"`
		CreatedAt    int64  `db:"created_at"`
	}
	var rows []fileRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, bucket, object_key, original_name, ext, mime, size, visibility, ref_count, COALESCE(uploader_id,0), UNIX_TIMESTAMP(created_at) FROM sys_file WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	list := make([]*filepb.FileInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &filepb.FileInfo{
			Id: r.Id, Bucket: r.Bucket, ObjectKey: r.ObjectKey, OriginalName: r.OriginalName,
			Ext: r.Ext, Mime: r.Mime, Size: r.Size, Visibility: r.Visibility,
			RefCount: r.RefCount, UploaderID: r.UploaderId, CreatedAt: r.CreatedAt,
		})
	}
	return &filepb.GetFileListResponse{List: list, Page: &filepb.PageResponse{Total: total}}, nil
}
