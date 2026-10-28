// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package appauth

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/appauth"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/pkg/result"
)

// 查询邮箱验证配置
func GetAppAuthOptionsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := appauth.NewGetAppAuthOptionsLogic(r.Context(), svcCtx)
		resp, err := l.GetAppAuthOptions()
		result.HttpResult(r, w, resp, err)
	}
}
