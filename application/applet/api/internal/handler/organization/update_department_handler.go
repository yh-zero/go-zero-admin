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

// 更新部门
func UpdateDepartmentHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.Department
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := organization.NewUpdateDepartmentLogic(r.Context(), svcCtx)
		resp, err := l.UpdateDepartment(&req)
		result.HttpResult(r, w, resp, err)
	}
}
