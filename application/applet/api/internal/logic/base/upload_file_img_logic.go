package base

import (
	"context"
	"io"
	"net/http"
	"strings"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

const maxFileSize = 10 << 20

type UploadFileImgLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUploadFileImgLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileImgLogic {
	return &UploadFileImgLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func readImage(r *http.Request) ([]byte, string, error) {
	if err := r.ParseMultipartForm(1 << 20); err != nil {
		return nil, "", xerr.NewErrCodeMsg(300002, "图片请求无效或超过10MB")
	}
	if r.MultipartForm != nil {
		defer r.MultipartForm.RemoveAll()
	}
	file, _, err := r.FormFile("file_img")
	if err != nil {
		return nil, "", xerr.NewErrCodeMsg(300002, "请选择图片，上传字段为file_img")
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
	if err != nil || len(data) == 0 || len(data) > maxFileSize {
		return nil, "", xerr.NewErrCodeMsg(300002, "图片不能为空且不能超过10MB")
	}
	mime := http.DetectContentType(data)
	switch mime {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return data, mime, nil
	}
	return nil, "", xerr.NewErrCodeMsg(300002, "仅支持PNG、JPEG、GIF、WebP图片")
}

// UploadFileImg is now a thin pass-through to the RPC tier. The RPC owns the
// OSS credentials, the rate-limited signing key, and the metadata registration
// in a single transaction. The API only parses the multipart request and
// forwards the bytes over gRPC.
func (l *UploadFileImgLogic) UploadFileImg(_ *types.UploadFileImgRequest, r *http.Request) (*types.UploadFileImgResponse, error) {
	data, mime, err := readImage(r)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.AppletFileRPC == nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件资源服务未配置")
	}
	visibility := strings.TrimSpace(r.FormValue("visibility"))
	if visibility == "" {
		visibility = "public"
	}
	if visibility != "public" && visibility != "private" {
		return nil, xerr.NewErrCodeMsg(300002, "文件可见性只允许public或private")
	}
	filename := ""
	if r.MultipartForm != nil && len(r.MultipartForm.File["file_img"]) > 0 {
		filename = r.MultipartForm.File["file_img"][0].Filename
	}
	actor := ctxJwt.GetJwtData(l.ctx)
	if actor.ID <= 0 || actor.AuthorityId <= 0 || actor.SessionVersion <= 0 {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
	}
	resp, err := l.svcCtx.AppletFileRPC.UploadFile(l.ctx, &pb.UploadFileRequest{
		Actor:      &pb.SessionRequest{UserID: actor.ID, AuthorityId: actor.AuthorityId, SessionVersion: actor.SessionVersion, SessionID: actor.SessionID},
		Data:       data,
		Mime:       mime,
		Filename:   filename,
		Visibility: visibility,
	})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.File == nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件资源服务响应无效")
	}
	return &types.UploadFileImgResponse{FileImgUrl: resp.Url, FileId: resp.File.ID}, nil
}
