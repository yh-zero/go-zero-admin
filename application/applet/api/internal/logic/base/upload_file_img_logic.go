package base

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
	"unicode/utf8"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
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
	visibility := r.FormValue("visibility")
	if visibility == "" {
		visibility = "public"
	}
	if visibility != "public" && visibility != "private" {
		return nil, xerr.NewErrCodeMsg(300002, "文件可见性只允许public或private")
	}
	name := path.Base(strings.ReplaceAll(r.MultipartForm.File["file_img"][0].Filename, "\\", "/"))
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || len([]rune(name)) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, "\x00\r\n") {
		return nil, xerr.NewErrCodeMsg(300002, "文件名无效或超过255字符")
	}
	if l.svcCtx.AppletFileRPC == nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件资源服务未配置")
	}
	actor := ctxJwt.GetJwtData(l.ctx)
	if actor.ID <= 0 || actor.AuthorityId <= 0 || actor.SessionVersion <= 0 {
		return nil, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR)
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
	acl := oss.ACLPublicRead
	if visibility == "private" {
		acl = oss.ACLPrivate
	}
	if err = bucket.PutObject(key, bytes.NewReader(data), oss.ContentType(mime), oss.ObjectACL(acl), oss.WithContext(l.ctx)); err != nil {
		l.Error("图片上传失败")
		return nil, xerr.NewErrCodeMsg(300002, "图片上传失败，请检查文件存储服务")
	}
	registration := &pb.RegisterFileRequest{Actor: &pb.SessionRequest{UserID: actor.ID, AuthorityId: actor.AuthorityId, SessionVersion: actor.SessionVersion, SessionID: actor.SessionID},
		File: &pb.FileResource{ObjectKey: key, Name: name, Mime: mime, Size: int64(len(data)), Visibility: visibility}}
	file, err := l.svcCtx.AppletFileRPC.RegisterFile(l.ctx, registration)
	if err != nil {
		// Retry the same idempotent registration after a transport ambiguity.
		// Never delete the blob here: the first RPC may have already committed.
		retryCtx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 2*time.Second)
		file, err = l.svcCtx.AppletFileRPC.RegisterFile(retryCtx, registration)
		cancel()
		if err != nil {
			l.Errorf("uploaded object requires metadata reconciliation key=%s", key)
			return nil, xerr.NewErrCodeMsg(300002, "图片已上传，但资源登记失败，请联系管理员核查")
		}
	}
	link := genFileURL(l.svcCtx.Config.Oss.BucketName, l.svcCtx.Config.Oss.Endpoint, key)
	if visibility == "private" {
		link, err = bucket.SignURL(key, oss.HTTPGet, 300)
		if err != nil {
			return nil, xerr.NewErrCodeMsg(300002, "图片已保存，访问地址生成失败")
		}
	}
	return &types.UploadFileImgResponse{FileImgUrl: link, FileId: file.ID}, nil
}
func genFileURL(bucketName, endpoint, key string) string {
	endpoint = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://"), "/")
	return (&url.URL{Scheme: "https", Host: bucketName + "." + endpoint, Path: "/" + key}).String()
}
