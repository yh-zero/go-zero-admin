// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package menu

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/applet/api/internal/logic/menu"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"
)

// 保存指定角色按钮授权
func UpdateAuthorityButtonsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UpdateAuthorityButtonsRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := menu.NewUpdateAuthorityButtonsLogic(r.Context(), svcCtx)
		resp, err := l.UpdateAuthorityButtons(&req)
		result.HttpResult(r, w, resp, err)
	}
}
