// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package menu

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/menu"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func PreviewMenuMoveHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.PreviewMenuMoveRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := menu.NewPreviewMenuMoveLogic(r.Context(), svcCtx)
		resp, err := l.PreviewMenuMove(&req)
		result.HttpResult(r, w, resp, err)
	}
}
