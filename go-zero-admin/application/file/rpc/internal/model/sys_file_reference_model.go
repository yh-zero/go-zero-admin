package model

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysFileReferenceModel = (*customSysFileReferenceModel)(nil)

type (
	// SysFileReferenceModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysFileReferenceModel.
	SysFileReferenceModel interface {
		sysFileReferenceModel
		AddReference(ctx context.Context, fileId, actorId int64, refType string, refId int64) error
		RemoveReference(ctx context.Context, fileId, actorId int64, refType string, refId int64) error
		ReconcileReferences(ctx context.Context, actorId int64, refType string, refId, keepFileId int64) error
	}

	customSysFileReferenceModel struct {
		*defaultSysFileReferenceModel
		conn sqlx.SqlConn
	}
)

// 调用方持有业务用户行锁；这里不反向调用业务服务，始终沿用 file→reference 的加锁顺序。
// 中途失败保留未处理明细，重复调用继续清理，不根据数量覆盖历史引用计数。
func (m *customSysFileReferenceModel) ReconcileReferences(ctx context.Context, actorId int64, refType string, refId, keepFileId int64) error {
	if actorId <= 0 {
		return ErrFileForbidden
	}
	var rows []struct {
		FileId int64 `db:"file_id"`
	}
	if err := m.QueryRowsNoCacheCtx(ctx, &rows, "SELECT file_id FROM sys_file_reference WHERE ref_type=? AND ref_id=? AND file_id<>? ORDER BY file_id", refType, refId, keepFileId); err != nil {
		return err
	}
	for _, row := range rows {
		if err := m.RemoveReference(ctx, row.FileId, actorId, refType, refId); err != nil {
			return err
		}
	}
	return nil
}

// NewSysFileReferenceModel returns a model for the database table.
func NewSysFileReferenceModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysFileReferenceModel {
	return &customSysFileReferenceModel{
		defaultSysFileReferenceModel: newSysFileReferenceModel(conn, c, opts...),
		conn:                         conn,
	}
}

// AddReference 与删除文件使用同一文件行锁，避免检查引用后被并发删除。
func (m *customSysFileReferenceModel) AddReference(ctx context.Context, fileId, actorId int64, refType string, refId int64) error {
	var reference SysFileReference
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var file SysFile
		if err := session.QueryRowCtx(ctx, &file, "SELECT "+sysFileRows+" FROM sys_file WHERE id=? AND deleted_at IS NULL FOR UPDATE", fileId); err != nil {
			return err
		}
		if !file.UploaderId.Valid || file.UploaderId.Int64 != actorId {
			return ErrFileForbidden
		}
		if !strings.HasPrefix(file.ObjectKey, "avatar/") || (file.Mime != "image/jpeg" && file.Mime != "image/png" && file.Mime != "image/gif") {
			return ErrFileNotAvatar
		}
		err := session.QueryRowCtx(ctx, &reference, "SELECT "+sysFileReferenceRows+" FROM sys_file_reference WHERE file_id=? AND ref_type=? AND ref_id=? FOR UPDATE", fileId, refType, refId)
		if err == nil {
			return nil
		}
		if !errors.Is(err, ErrNotFound) {
			return err
		}
		result, err := session.ExecCtx(ctx, "INSERT INTO sys_file_reference (file_id,ref_type,ref_id,created_by) VALUES (?,?,?,?)", fileId, refType, refId, actorId)
		if err != nil {
			return err
		}
		id, err := result.LastInsertId()
		if err != nil {
			return err
		}
		reference.Id = uint64(id)
		_, err = session.ExecCtx(ctx, "UPDATE sys_file SET ref_count=ref_count+1 WHERE id=?", fileId)
		return err
	})
	if err != nil {
		return err
	}
	return m.invalidateReference(ctx, fileId, reference.Id, refType, refId)
}

// RemoveReference 无明细则不减计数，保留历史无法追溯引用的保护。
func (m *customSysFileReferenceModel) RemoveReference(ctx context.Context, fileId, actorId int64, refType string, refId int64) error {
	if actorId <= 0 {
		return ErrFileForbidden
	}
	var reference SysFileReference
	err := m.conn.TransactCtx(ctx, func(ctx context.Context, session sqlx.Session) error {
		var file SysFile
		if err := session.QueryRowCtx(ctx, &file, "SELECT "+sysFileRows+" FROM sys_file WHERE id=? FOR UPDATE", fileId); err != nil {
			return err
		}
		err := session.QueryRowCtx(ctx, &reference, "SELECT "+sysFileReferenceRows+" FROM sys_file_reference WHERE file_id=? AND ref_type=? AND ref_id=? FOR UPDATE", fileId, refType, refId)
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		if err != nil {
			return err
		}
		// 仅供已完成目标对象授权的业务 RPC 调用；旧 fileId 必须由调用方从数据库取得。
		if _, err := session.ExecCtx(ctx, "DELETE FROM sys_file_reference WHERE id=?", reference.Id); err != nil {
			return err
		}
		// 只释放这一条确知存在的引用，不按当前明细数覆盖历史计数。
		_, err = session.ExecCtx(ctx, "UPDATE sys_file SET ref_count=GREATEST(ref_count-1,0) WHERE id=?", fileId)
		return err
	})
	if err != nil {
		return err
	}
	return m.invalidateReference(ctx, fileId, reference.Id, refType, refId)
}

func (m *customSysFileReferenceModel) invalidateReference(ctx context.Context, fileId int64, referenceId uint64, refType string, refId int64) error {
	keys := []string{
		fmt.Sprintf("%s%d", cacheGozeroAdminSysFileIdPrefix, fileId),
		fmt.Sprintf("%s%d:%s:%d", cacheGozeroAdminSysFileReferenceFileIdRefTypeRefIdPrefix, fileId, refType, refId),
	}
	if referenceId > 0 {
		keys = append(keys, fmt.Sprintf("%s%d", cacheGozeroAdminSysFileReferenceIdPrefix, referenceId))
	}
	return m.DelCacheCtx(ctx, keys...)
}
