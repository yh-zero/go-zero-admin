package model

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysFileModel = (*customSysFileModel)(nil)

type (
	// SysFileModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysFileModel.
	SysFileModel interface {
		sysFileModel
		// ListFilePage 包含有效文件及当前上传者尚未完成清理的文件。
		ListFilePage(ctx context.Context, f FileListFilter, page, pageSize int64) ([]*FileRow, int64, error)
		FindLive(ctx context.Context, id int64) (*SysFile, error)
		MarkDeletedIfUnused(ctx context.Context, id, actorId int64) (*SysFile, error)
		ConfirmObjectCleanup(ctx context.Context, id int64) error
		GetFileStats(ctx context.Context, viewerId int64) (*FileStats, error)
	}

	customSysFileModel struct {
		*defaultSysFileModel
		conn sqlx.SqlConn
	}
)

// FileListFilter 文件列表筛选（空值/0 不过滤）
type FileListFilter struct {
	Visibility         string
	UploaderId         int64
	ViewerId           int64
	CleanupPendingOnly bool
	Name               string
	Mime               string
	Ext                string
	MinSize            int64
	MaxSize            int64
	StartTime          int64
	EndTime            int64
}

// FileRow 文件行（uploader_id 可空 COALESCE；时间列转 int64）
type FileRow struct {
	Id             int64  `db:"id"`
	Bucket         string `db:"bucket"`
	ObjectKey      string `db:"object_key"`
	OriginalName   string `db:"original_name"`
	Ext            string `db:"ext"`
	Mime           string `db:"mime"`
	Size           int64  `db:"size"`
	Visibility     string `db:"visibility"`
	RefCount       int32  `db:"ref_count"`
	UploaderId     int64  `db:"uploader_id"`
	UploaderName   string `db:"uploader_name"`
	SHA256         string `db:"sha256"`
	CreatedAt      int64  `db:"created_at"`
	CleanupPending bool   `db:"cleanup_pending"`
}

type FileStats struct {
	TotalFiles          int64 `db:"total_files"`
	TotalBytes          int64 `db:"total_bytes"`
	PublicFiles         int64 `db:"public_files"`
	PublicBytes         int64 `db:"public_bytes"`
	PrivateFiles        int64 `db:"private_files"`
	PrivateBytes        int64 `db:"private_bytes"`
	CleanupPendingFiles int64 `db:"cleanup_pending_files"`
	CleanupPendingBytes int64 `db:"cleanup_pending_bytes"`
}

// NewSysFileModel returns a model for the database table.
func NewSysFileModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysFileModel {
	return &customSysFileModel{
		defaultSysFileModel: newSysFileModel(conn, c, opts...),
		conn:                conn,
	}
}

var (
	ErrFileForbidden     = errors.New("file access denied")
	ErrFileReferenced    = errors.New("file is referenced")
	ErrFileNotAvatar     = errors.New("file is not an avatar")
	ErrInvalidFileFilter = errors.New("invalid file list filter")
)

var fileNameEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

// FindLive 不缓存授权相关状态，避免删除或引用事务后的旧缓存扩大访问范围。
func (m *customSysFileModel) FindLive(ctx context.Context, id int64) (*SysFile, error) {
	var row SysFile
	err := m.QueryRowNoCacheCtx(ctx, &row, "SELECT "+sysFileRows+" FROM sys_file WHERE id=? AND deleted_at IS NULL", id)
	return &row, err
}

// MarkDeletedIfUnused 先持久化删除状态；对象存储失败后重复删除可继续清理，期间拒绝新增引用。
func (m *customSysFileModel) MarkDeletedIfUnused(ctx context.Context, id, actorId int64) (*SysFile, error) {
	var row SysFile
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		if err := session.QueryRowCtx(ctx, &row, "SELECT "+sysFileRows+" FROM sys_file WHERE id=? FOR UPDATE", id); err != nil {
			return err
		}
		if !row.UploaderId.Valid || row.UploaderId.Int64 != actorId {
			return ErrFileForbidden
		}
		var count struct {
			Total int64 `db:"total"`
		}
		if err := session.QueryRowCtx(ctx, &count, "SELECT COUNT(*) AS total FROM sys_file_reference WHERE file_id=?", id); err != nil {
			return err
		}
		if row.RefCount != 0 || count.Total != 0 {
			return ErrFileReferenced
		}
		if row.DeletedAt.Valid {
			return nil
		}
		_, err := session.ExecCtx(ctx, "UPDATE sys_file SET deleted_at=NOW() WHERE id=?", id)
		return err
	})
	if err != nil {
		return nil, err
	}
	if err := m.DelCacheCtx(ctx, fmt.Sprintf("%s%d", cacheGozeroAdminSysFileIdPrefix, id)); err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *customSysFileModel) ConfirmObjectCleanup(ctx context.Context, id int64) error {
	_, err := m.conn.ExecCtx(ctx, "UPDATE sys_file SET object_cleaned_at=NOW() WHERE id=? AND deleted_at IS NOT NULL AND object_cleaned_at IS NULL", id)
	return err
}

func fileListWhere(f FileListFilter) (string, []any, error) {
	if f.MinSize < 0 || f.MaxSize < 0 || f.StartTime < 0 || f.EndTime < 0 ||
		(f.MinSize > 0 && f.MaxSize > 0 && f.MinSize > f.MaxSize) ||
		(f.StartTime > 0 && f.EndTime > 0 && f.StartTime > f.EndTime) {
		return "", nil, ErrInvalidFileFilter
	}
	where := "f.deleted_at IS NULL"
	var args []any
	if f.ViewerId > 0 {
		where = "(f.deleted_at IS NULL OR (f.uploader_id=? AND f.object_cleaned_at IS NULL))"
		args = append(args, f.ViewerId)
	}
	if f.CleanupPendingOnly {
		if f.ViewerId <= 0 {
			return "", nil, ErrInvalidFileFilter
		}
		where = "f.deleted_at IS NOT NULL AND f.uploader_id=? AND f.object_cleaned_at IS NULL"
		args = []any{f.ViewerId}
	}
	if f.Visibility != "" {
		where += " AND f.visibility=?"
		args = append(args, f.Visibility)
	}
	if f.UploaderId > 0 {
		where += " AND f.uploader_id=?"
		args = append(args, f.UploaderId)
	}
	if f.Name != "" {
		where += " AND f.original_name LIKE ? ESCAPE '!'"
		args = append(args, "%"+fileNameEscaper.Replace(f.Name)+"%")
	}
	if f.Mime != "" {
		where += " AND f.mime=?"
		args = append(args, f.Mime)
	}
	if f.Ext != "" {
		where += " AND f.ext=?"
		args = append(args, f.Ext)
	}
	if f.MinSize > 0 {
		where += " AND f.size>=?"
		args = append(args, f.MinSize)
	}
	if f.MaxSize > 0 {
		where += " AND f.size<=?"
		args = append(args, f.MaxSize)
	}
	if f.StartTime > 0 {
		where += " AND f.created_at>=FROM_UNIXTIME(?)"
		args = append(args, f.StartTime)
	}
	if f.EndTime > 0 {
		where += " AND f.created_at<=FROM_UNIXTIME(?)"
		args = append(args, f.EndTime)
	}
	return where, args, nil
}

func (m *customSysFileModel) ListFilePage(ctx context.Context, f FileListFilter, page, pageSize int64) ([]*FileRow, int64, error) {
	where, args, err := fileListWhere(f)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_file f WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	order := "f.id DESC"
	if f.ViewerId > 0 && !f.CleanupPendingOnly {
		order = "f.deleted_at IS NOT NULL DESC, f.id DESC"
	}
	var rows []*FileRow
	err = m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT f.id, f.bucket, f.object_key, f.original_name, f.ext, f.mime, f.size, f.sha256, f.visibility, f.ref_count, "+
			"COALESCE(f.uploader_id,0) AS uploader_id, COALESCE(u.username,'') AS uploader_name, UNIX_TIMESTAMP(f.created_at) AS created_at, (f.deleted_at IS NOT NULL) AS cleanup_pending "+
			"FROM sys_file f LEFT JOIN sys_user u ON u.id=f.uploader_id WHERE "+where+" ORDER BY "+order+" LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}

func (m *customSysFileModel) GetFileStats(ctx context.Context, viewerId int64) (*FileStats, error) {
	if viewerId <= 0 {
		return nil, ErrFileForbidden
	}
	var stats FileStats
	err := m.QueryRowNoCacheCtx(ctx, &stats, `SELECT
		COALESCE(SUM(f.deleted_at IS NULL),0) AS total_files,
		COALESCE(SUM(CASE WHEN f.deleted_at IS NULL THEN f.size ELSE 0 END),0) AS total_bytes,
		COALESCE(SUM(f.deleted_at IS NULL AND f.visibility='public'),0) AS public_files,
		COALESCE(SUM(CASE WHEN f.deleted_at IS NULL AND f.visibility='public' THEN f.size ELSE 0 END),0) AS public_bytes,
		COALESCE(SUM(f.deleted_at IS NULL AND f.visibility='private'),0) AS private_files,
		COALESCE(SUM(CASE WHEN f.deleted_at IS NULL AND f.visibility='private' THEN f.size ELSE 0 END),0) AS private_bytes,
		COALESCE(SUM(f.deleted_at IS NOT NULL AND f.object_cleaned_at IS NULL),0) AS cleanup_pending_files,
		COALESCE(SUM(CASE WHEN f.deleted_at IS NOT NULL AND f.object_cleaned_at IS NULL THEN f.size ELSE 0 END),0) AS cleanup_pending_bytes
		FROM sys_file f WHERE f.uploader_id=?`, viewerId)
	return &stats, err
}
