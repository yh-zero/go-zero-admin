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

// 按数据范围分页查询文件
func GetFileListHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.FileListRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := files.NewGetFileListLogic(r.Context(), svcCtx)
		resp, err := l.GetFileList(&req)
		result.HttpResult(r, w, resp, err)
	}
}
