package model

import (
	"context"
	"errors"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SysAuditLogModel = (*customSysAuditLogModel)(nil)

var ErrInvalidAuditFilter = errors.New("invalid audit log filter")
var auditPathEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

type (
	SysAuditLogModel interface {
		sysAuditLogModel
		ListAuditLogPage(ctx context.Context, f AuditLogListFilter, page, pageSize int64) ([]*AuditLogRow, int64, error)
		PurgeBefore(ctx context.Context, cutoffUnix int64, batchSize int) (int64, error)
	}

	customSysAuditLogModel struct {
		*defaultSysAuditLogModel
	}
)

// AuditLogListFilter 审计列表筛选（空值/0 不过滤）。
type AuditLogListFilter struct {
	Module        string
	Action        string
	UserId        int64
	Username      string
	Method        string
	Path          string
	IP            string
	Result        int64
	StartTime     int64
	EndTime       int64
	MinDurationMs int64
	MaxDurationMs int64
}

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

func NewSysAuditLogModel(conn sqlx.SqlConn, c cache.CacheConf, opts ...cache.Option) SysAuditLogModel {
	return &customSysAuditLogModel{defaultSysAuditLogModel: newSysAuditLogModel(conn, c, opts...)}
}

func auditLogWhere(f AuditLogListFilter) (string, []any, error) {
	if f.StartTime < 0 || f.EndTime < 0 || f.MinDurationMs < 0 || f.MaxDurationMs < 0 ||
		(f.StartTime > 0 && f.EndTime > 0 && f.StartTime > f.EndTime) ||
		(f.MinDurationMs > 0 && f.MaxDurationMs > 0 && f.MinDurationMs > f.MaxDurationMs) {
		return "", nil, ErrInvalidAuditFilter
	}
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
	if f.Username != "" {
		where += " AND username=?"
		args = append(args, f.Username)
	}
	if f.Method != "" {
		where += " AND method=?"
		args = append(args, f.Method)
	}
	if f.Path != "" {
		where += " AND path LIKE ? ESCAPE '!'"
		args = append(args, "%"+auditPathEscaper.Replace(f.Path)+"%")
	}
	if f.IP != "" {
		where += " AND ip=?"
		args = append(args, f.IP)
	}
	if f.Result > 0 {
		where += " AND result=?"
		args = append(args, f.Result)
	}
	if f.StartTime > 0 {
		where += " AND created_at>=FROM_UNIXTIME(?)"
		args = append(args, f.StartTime)
	}
	if f.EndTime > 0 {
		where += " AND created_at<=FROM_UNIXTIME(?)"
		args = append(args, f.EndTime)
	}
	if f.MinDurationMs > 0 {
		where += " AND duration_ms>=?"
		args = append(args, f.MinDurationMs)
	}
	if f.MaxDurationMs > 0 {
		where += " AND duration_ms<=?"
		args = append(args, f.MaxDurationMs)
	}
	return where, args, nil
}

func (m *customSysAuditLogModel) ListAuditLogPage(ctx context.Context, f AuditLogListFilter, page, pageSize int64) ([]*AuditLogRow, int64, error) {
	where, args, err := auditLogWhere(f)
	if err != nil {
		return nil, 0, err
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
	err = m.QueryRowsNoCacheCtx(ctx, &rows,
		"SELECT id, COALESCE(user_id,0) AS user_id, username, module, action, method, path, COALESCE(params,'') AS params, "+
			"result, error_msg, ip, user_agent, duration_ms, UNIX_TIMESTAMP(created_at) AS created_at "+
			"FROM sys_audit_log WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, (page-1)*pageSize)...)
	return rows, total, err
}

// PurgeBefore 每批限制删除行数，由调用方控制频率，避免长事务和大批删除。
func (m *customSysAuditLogModel) PurgeBefore(ctx context.Context, cutoffUnix int64, batchSize int) (int64, error) {
	if cutoffUnix <= 0 || batchSize <= 0 {
		return 0, ErrInvalidAuditFilter
	}
	result, err := m.ExecNoCacheCtx(ctx, "DELETE FROM sys_audit_log WHERE created_at<FROM_UNIXTIME(?) LIMIT ?", cutoffUnix, batchSize)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
