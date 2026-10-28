package dictionary

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/dictionary"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func DeleteSysDictionaryInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.DeleteSysDictionaryInfoRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := dictionary.NewDeleteSysDictionaryInfoLogic(r.Context(), svcCtx)
		resp, err := l.DeleteSysDictionaryInfo(&req)
		result.HttpResult(r, w, resp, err)
	}
}
