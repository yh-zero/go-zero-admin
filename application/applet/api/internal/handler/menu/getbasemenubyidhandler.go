package menu

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/menu"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetBaseMenuByIdHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetBaseMenuByIdRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := menu.NewGetBaseMenuByIdLogic(r.Context(), svcCtx)
		resp, err := l.GetBaseMenuById(&req)
		result.HttpResult(r, w, resp, err)
	}
}
