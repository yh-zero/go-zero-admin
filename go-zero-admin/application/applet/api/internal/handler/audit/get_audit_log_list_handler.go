package audit

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	auditlogic "go-zero-admin/application/applet/api/internal/logic/audit"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"
)

func GetAuditLogListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetAuditLogListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}
		response, err := auditlogic.NewGetAuditLogListLogic(r.Context(), svcCtx).GetAuditLogList(&req)
		result.HttpResult(r, w, response, err)
	}
}
