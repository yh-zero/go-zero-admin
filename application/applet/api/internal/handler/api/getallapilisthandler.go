package api

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/api"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetAllApiListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetAllApiListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := api.NewGetAllApiListLogic(r.Context(), svcCtx)
		resp, err := l.GetAllApiList(&req)
		result.HttpResult(r, w, resp, err)
	}
}
