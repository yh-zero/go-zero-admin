// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package files

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/files"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 移除文件业务引用
func RemoveFileReferenceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileReferenceRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := files.NewRemoveFileReferenceLogic(r.Context(), svcCtx)
		resp, err := l.RemoveFileReference(&req)
		result.HttpResult(r, w, resp, err)
	}
}
