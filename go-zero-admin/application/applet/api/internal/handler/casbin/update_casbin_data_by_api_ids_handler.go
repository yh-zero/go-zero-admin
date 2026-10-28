package casbin

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/casbin"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateCasbinDataByApiIdsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateCasbinDataByApiIdsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := casbin.NewUpdateCasbinDataByApiIdsLogic(r.Context(), svcCtx)
		resp, err := l.UpdateCasbinDataByApiIds(&req)
		result.HttpResult(r, w, resp, err)
	}
}
