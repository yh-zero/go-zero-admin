// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package devicesession

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/devicesession"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 管理员查询有效设备会话
func GetDeviceSessionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AdminDeviceSessionListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := devicesession.NewGetDeviceSessionsLogic(r.Context(), svcCtx)
		resp, err := l.GetDeviceSessions(&req)
		result.HttpResult(r, w, resp, err)
	}
}
