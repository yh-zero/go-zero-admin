// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package dict

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/logic/dict"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
)

func DeleteDictHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteDictReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := dict.NewDeleteDictLogic(r.Context(), svcCtx)
		err := l.DeleteDict(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.Ok(w)
		}
	}
}
