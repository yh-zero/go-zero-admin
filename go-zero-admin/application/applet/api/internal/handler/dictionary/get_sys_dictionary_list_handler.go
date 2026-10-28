package dictionary

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/dictionary"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetSysDictionaryListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetSysDictionaryListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := dictionary.NewGetSysDictionaryListLogic(r.Context(), svcCtx)
		resp, err := l.GetSysDictionaryList(&req)
		result.HttpResult(r, w, resp, err)
	}
}
