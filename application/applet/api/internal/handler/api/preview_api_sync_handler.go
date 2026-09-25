// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package api

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/api"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/pkg/result"
)

// 预览Swagger与API资源差异，失效资源仅提示
func PreviewApiSyncHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := api.NewPreviewApiSyncLogic(r.Context(), svcCtx)
		resp, err := l.PreviewApiSync()
		result.HttpResult(r, w, resp, err)
	}
}
