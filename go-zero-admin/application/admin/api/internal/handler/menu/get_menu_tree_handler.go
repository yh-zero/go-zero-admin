// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package menu

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/logic/menu"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
)

func GetMenuTreeHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetMenuTreeReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := menu.NewGetMenuTreeLogic(r.Context(), svcCtx)
		resp, err := l.GetMenuTree(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
