package user

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/user"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func ResetUserPasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ResetUserPasswordRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := user.NewResetUserPasswordLogic(r.Context(), svcCtx)
		resp, err := l.ResetUserPassword(&req)
		result.HttpResult(r, w, resp, err)
	}
}
