package audit

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/logic/audit"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
)

var auditCSVHeader = []string{"id", "user_id", "username", "module", "action", "method", "path", "result", "error_msg", "ip", "user_agent", "duration_ms", "created_at"}

func ExportAuditLogsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetAuditLogListReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		rows, err := audit.NewExportAuditLogsLogic(r.Context(), svcCtx).ExportAuditLogs(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename=\"audit-logs.csv\"`)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		_, _ = w.Write([]byte{0xef, 0xbb, 0xbf})
		writer := csv.NewWriter(w)
		_ = writer.Write(auditCSVHeader)
		for _, row := range rows {
			_ = writer.Write([]string{
				strconv.FormatInt(row.Id, 10), strconv.FormatInt(row.UserId, 10),
				csvSafe(row.Username), csvSafe(row.Module), csvSafe(row.Action), csvSafe(row.Method), csvSafe(row.Path),
				strconv.FormatInt(int64(row.Result), 10), csvSafe(row.ErrorMsg), csvSafe(row.Ip), csvSafe(row.UserAgent),
				strconv.FormatInt(int64(row.DurationMs), 10), time.Unix(row.CreatedAt, 0).In(time.FixedZone("Asia/Shanghai", 8*3600)).Format("2006-01-02 15:04:05"),
			})
		}
		writer.Flush()
	}
}

// csvSafe 防止单元格被表格软件解释为公式；CSV 库负责逗号、引号及换行转义。
func csvSafe(value string) string {
	trimmed := strings.TrimLeft(value, " 	")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@', '	', '':
		return "'" + value
	default:
		return value
	}
}
