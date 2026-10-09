// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package permission

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/permission"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func PreviewPermissionRollbackHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PreviewPermissionRollbackRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := permission.NewPreviewPermissionRollbackLogic(r.Context(), svcCtx)
		resp, err := l.PreviewPermissionRollback(&req)
		result.HttpResult(r, w, resp, err)
	}
}
