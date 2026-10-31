package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysFileModel = (*customSysFileModel)(nil)

type (
	// SysFileModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysFileModel.
	SysFileModel interface {
		sysFileModel
		// ListFilePage 分页查有效文件
		ListFilePage(ctx context.Context, f FileListFilter, page, pageSize int64) ([]*FileRow, int64, error)
	}

	customSysFileModel struct {
		*defaultSysFileModel
	}
)

// FileListFilter 文件列表筛选（空值/0 不过滤）
type FileListFilter struct {
	Visibility string
	UploaderId int64
}

// FileRow 文件行（uploader_id 可空 COALESCE；时间列转 int64）
type FileRow struct {
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

// NewSysFileModel returns a model for the database table.
func NewSysFileModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysFileModel {
	return &customSysFileModel{
		defaultSysFileModel: newSysFileModel(conn, c, opts...),
	}
}

func (m *customSysFileModel) ListFilePage(ctx context.Context, f FileListFilter, page, pageSize int64) ([]*FileRow, int64, error) {
	where := "deleted_at IS NULL"
	var args []any
	if f.Visibility != "" {
		where += " AND visibility=?"
		args = append(args, f.Visibility)
	}
	if f.UploaderId > 0 {
		where += " AND uploader_id=?"
		args = append(args, f.UploaderId)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_file WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*FileRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, bucket, object_key, original_name, ext, mime, size, visibility, ref_count, "+
			"COALESCE(uploader_id,0) AS uploader_id, UNIX_TIMESTAMP(created_at) AS created_at "+
			"FROM sys_file WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}
