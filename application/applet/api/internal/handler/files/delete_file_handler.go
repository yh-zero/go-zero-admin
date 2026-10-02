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

// 删除无引用文件，失败可重试
func DeleteFileHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileIDRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := files.NewDeleteFileLogic(r.Context(), svcCtx)
		resp, err := l.DeleteFile(&req)
		result.HttpResult(r, w, resp, err)
	}
}
