package base

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result/xerr"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gofrs/uuid/v5"
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
	if imageExtension(mime) == "" {
		return nil, "", xerr.NewErrCodeMsg(300002, "仅支持PNG、JPEG、GIF、WebP图片")
	}
	return data, mime, nil
}
func imageExtension(mime string) string {
	switch mime {
	case "image/png":
		return ".png"
	case "image/jpeg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	}
	return ""
}
func (l *UploadFileImgLogic) UploadFileImg(_ *types.UploadFileImgRequest, r *http.Request) (*types.UploadFileImgResponse, error) {
	data, mime, err := readImage(r)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.OssClient == nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务未配置")
	}
	bucket, err := l.svcCtx.OssClient.Bucket(l.svcCtx.Config.Oss.BucketName)
	if err != nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务配置无效")
	}
	id, err := uuid.NewV4()
	if err != nil {
		return nil, err
	}
	key := "go-zero-admin/" + id.String() + imageExtension(mime)
	if err = bucket.PutObject(key, bytes.NewReader(data), oss.ContentType(mime), oss.WithContext(l.ctx)); err != nil {
		l.Error("图片上传失败")
		return nil, xerr.NewErrCodeMsg(300002, "图片上传失败，请检查文件存储服务")
	}
	return &types.UploadFileImgResponse{FileImgUrl: genFileURL(l.svcCtx.Config.Oss.BucketName, l.svcCtx.Config.Oss.Endpoint, key)}, nil
}
func genFileURL(bucketName, endpoint, key string) string {
	endpoint = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://"), "/")
	return (&url.URL{Scheme: "https", Host: bucketName + "." + endpoint, Path: "/" + key}).String()
}
