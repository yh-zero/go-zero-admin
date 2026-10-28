package base

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/base"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"
)

func UploadFileImgHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.UploadFileImgRequest
		// Bound the whole request before parsing multipart data, including its overhead.
		r.Body = http.MaxBytesReader(w, r.Body, 11<<20)

		l := base.NewUploadFileImgLogic(r.Context(), svcCtx)
		resp, err := l.UploadFileImg(&req, r)
		result.HttpResult(r, w, resp, err)
	}
}
