package authority

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/authority"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func UpdateAuthorityHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateAuthorityRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := authority.NewUpdateAuthorityLogic(r.Context(), svcCtx)
		resp, err := l.UpdateAuthority(&req)
		result.HttpResult(r, w, resp, err)
	}
}
