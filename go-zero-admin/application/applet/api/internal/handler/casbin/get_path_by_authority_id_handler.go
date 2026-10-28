package casbin

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/casbin"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func GetPathByAuthorityIdHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.GetPathByAuthorityIdRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := casbin.NewGetPathByAuthorityIdLogic(r.Context(), svcCtx)
		resp, err := l.GetPathByAuthorityId(&req)
		result.HttpResult(r, w, resp, err)
	}
}
