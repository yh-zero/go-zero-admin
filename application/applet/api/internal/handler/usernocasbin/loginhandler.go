package usernocasbin

import (
	"context"
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/usernocasbin"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		ctx := context.WithValue(r.Context(), "RemoteAddr", r.RemoteAddr)
		ctx = context.WithValue(ctx, "Isdev", r.Header.Get("Isdev"))
		r = r.WithContext(ctx)

		l := usernocasbin.NewLoginLogic(r.Context(), svcCtx)
		resp, err := l.Login(&req)
		result.HttpResult(r, w, resp, err)
	}
}
