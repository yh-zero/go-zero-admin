package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysAuditLogModel = (*customSysAuditLogModel)(nil)

type (
	// SysAuditLogModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSysAuditLogModel.
	SysAuditLogModel interface {
		sysAuditLogModel
		// ListAuditLogPage 分页查审计日志
		ListAuditLogPage(ctx context.Context, f AuditLogListFilter, page, pageSize int64) ([]*AuditLogRow, int64, error)
	}

	customSysAuditLogModel struct {
		*defaultSysAuditLogModel
	}
)

// AuditLogListFilter 审计列表筛选（空值/0 不过滤）
type AuditLogListFilter struct {
	Module string
	Action string
	UserId int64
	Result int64
}

// AuditLogRow 审计行（user_id/params 可空，COALESCE 兜底；时间列转 int64）
type AuditLogRow struct {
	Id         int64  `db:"id"`
	UserId     int64  `db:"user_id"`
	Username   string `db:"username"`
	Module     string `db:"module"`
	Action     string `db:"action"`
	Method     string `db:"method"`
	Path       string `db:"path"`
	Params     string `db:"params"`
	Result     int32  `db:"result"`
	ErrorMsg   string `db:"error_msg"`
	IP         string `db:"ip"`
	UserAgent  string `db:"user_agent"`
	DurationMs int32  `db:"duration_ms"`
	CreatedAt  int64  `db:"created_at"`
}

// NewSysAuditLogModel returns a model for the database table.
func NewSysAuditLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysAuditLogModel {
	return &customSysAuditLogModel{
		defaultSysAuditLogModel: newSysAuditLogModel(conn, c, opts...),
	}
}

func (m *customSysAuditLogModel) ListAuditLogPage(ctx context.Context, f AuditLogListFilter, page, pageSize int64) ([]*AuditLogRow, int64, error) {
	where := "1=1"
	var args []any
	if f.Module != "" {
		where += " AND module=?"
		args = append(args, f.Module)
	}
	if f.Action != "" {
		where += " AND action=?"
		args = append(args, f.Action)
	}
	if f.UserId > 0 {
		where += " AND user_id=?"
		args = append(args, f.UserId)
	}
	if f.Result > 0 {
		where += " AND result=?"
		args = append(args, f.Result)
	}
	var total int64
	if err := m.QueryRowNoCacheCtx(ctx, &total, "SELECT COUNT(*) FROM sys_audit_log WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	var rows []*AuditLogRow
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, COALESCE(user_id,0) AS user_id, username, module, action, method, path, COALESCE(params,'') AS params, "+
			"result, error_msg, ip, user_agent, duration_ms, UNIX_TIMESTAMP(created_at) AS created_at "+
			"FROM sys_audit_log WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}
