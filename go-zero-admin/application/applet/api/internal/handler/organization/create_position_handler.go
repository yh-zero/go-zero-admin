// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organization

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/organization"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 创建岗位
func CreatePositionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Position
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := organization.NewCreatePositionLogic(r.Context(), svcCtx)
		resp, err := l.CreatePosition(&req)
		result.HttpResult(r, w, resp, err)
	}
}
