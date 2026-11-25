package file

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/logic/file"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/pkg/xerr"
)

func UploadMultipartHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "multipart 表单不合法"))
			return
		}
		if r.MultipartForm == nil {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "multipart 表单不能为空"))
			return
		}
		defer func() { _ = r.MultipartForm.RemoveAll() }()
		if len(r.MultipartForm.File) != 1 || len(r.MultipartForm.File["file"]) != 1 {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "必须且只能上传一个 file 部件"))
			return
		}
		header := r.MultipartForm.File["file"][0]
		if header.Size <= 0 || header.Size > 10<<20 {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件大小须为 1 字节至 10 MiB"))
			return
		}
		content, err := header.Open()
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "无法读取上传文件"))
			return
		}
		defer content.Close()
		req := types.UploadMultipartReq{Scene: r.FormValue("scene"), Visibility: r.FormValue("visibility")}
		resp, err := file.NewUploadMultipartLogic(r.Context(), svcCtx).UploadMultipart(&req, header.Filename, content)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
		} else {
			httpx.OkJsonCtx(r.Context(), w, resp)
		}
	}
}
