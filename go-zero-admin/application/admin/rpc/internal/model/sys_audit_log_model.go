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
		FindList(context.Context, string, string, int64, int32, int64, int64) ([]AuditLogRow, int64, error)
	}

	customSysAuditLogModel struct {
		*defaultSysAuditLogModel
	}
)

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
	Ip         string `db:"ip"`
	UserAgent  string `db:"user_agent"`
	DurationMs int32  `db:"duration_ms"`
	CreatedAt  int64  `db:"created_at"`
}

func (m *customSysAuditLogModel) FindList(ctx context.Context, module, action string, userId int64, result int32, page, pageSize int64) ([]AuditLogRow, int64, error) {
	where := "1=1"
	var args []any
	if module != "" {
		where += " AND module=?"
		args = append(args, module)
	}
	if action != "" {
		where += " AND action=?"
		args = append(args, action)
	}
	if userId > 0 {
		where += " AND user_id=?"
		args = append(args, userId)
	}
	if result > 0 {
		where += " AND result=?"
		args = append(args, result)
	}
	var count totalRow
	if err := m.QueryRowNoCacheCtx(ctx, &count, "SELECT COUNT(*) AS total FROM sys_audit_log WHERE "+where, args...); err != nil {
		return nil, 0, err
	}
	limit, offset := pageBounds(page, pageSize)
	rows := make([]AuditLogRow, 0)
	err := m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, COALESCE(user_id,0) AS user_id, username, module, action, method, path, COALESCE(params,'') AS params, result, error_msg, ip, user_agent, duration_ms, UNIX_TIMESTAMP(created_at) AS created_at FROM sys_audit_log WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, limit, offset)...)
	return rows, count.Total, err
}

// NewSysAuditLogModel returns a model for the database table.
func NewSysAuditLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysAuditLogModel {
	return &customSysAuditLogModel{
		defaultSysAuditLogModel: newSysAuditLogModel(conn, c, opts...),
	}
}
