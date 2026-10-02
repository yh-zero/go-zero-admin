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

// 查询岗位列表
func GetPositionListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.OrganizationListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := organization.NewGetPositionListLogic(r.Context(), svcCtx)
		resp, err := l.GetPositionList(&req)
		result.HttpResult(r, w, resp, err)
	}
}
