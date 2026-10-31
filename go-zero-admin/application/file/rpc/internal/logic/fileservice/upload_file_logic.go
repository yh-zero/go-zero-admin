package fileservicelogic

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"path/filepath"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/application/file/rpc/internal/model"
	"go-zero-admin/application/file/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type UploadFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UploadFileLogic) UploadFile(in *filepb.UploadFileRequest) (*filepb.UploadFileResponse, error) {
	if in.GetActor().GetUserID() <= 0 {
		return nil, xerr.NewErrCode(xerr.PERMISSION_DENIED)
	}
	name, ext, mime, scene, visibility, err := validateUpload(in)
	if err != nil {
		return nil, err
	}
	if l.svcCtx.Store == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "对象存储未配置")
	}
	objectKey := buildObjectKey(scene, ext)
	if err := l.svcCtx.Store.Put(l.ctx, objectKey, mime, in.Content, visibility); err != nil {
		l.Error("文件对象写入失败")
		return nil, xerr.NewErrCodeMsg(500, "文件上传失败")
	}

	bucket := l.svcCtx.Config.Storage.PublicBucket
	if visibility == "private" {
		bucket = l.svcCtx.Config.Storage.PrivateBucket
	}
	size := int64(len(in.Content))

	result, err := l.svcCtx.SysFileModel.Insert(l.ctx, &model.SysFile{
		Bucket: bucket, ObjectKey: objectKey, OriginalName: name, Ext: ext, Mime: mime,
		Size: uint64(size), Visibility: visibility, UploaderId: sql.NullInt64{Int64: in.Actor.UserID, Valid: true},
	})
	if err != nil {
		l.Error("文件元数据写入失败")
		// 生成 model 在缓存删除失败时可能已提交；有结果时保留对象，避免制造失效记录。
		if result == nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(l.ctx), 10*time.Second)
			defer cancel()
			if cleanupErr := l.svcCtx.Store.Delete(cleanupCtx, objectKey, visibility); cleanupErr != nil {
				l.Error("文件上传补偿清理失败")
			}
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	// public 直接给 URL，private 需调 GetFileURL 获取签名
	var url string
	if visibility == "public" {
		url, _, err = l.svcCtx.Store.URL(l.ctx, objectKey, "public")
		if err != nil {
			return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
	}

	return &filepb.UploadFileResponse{File: &filepb.FileInfo{
		Id: id, Bucket: bucket, ObjectKey: objectKey, OriginalName: name,
		Ext: ext, Mime: mime, Size: size, Visibility: visibility,
		RefCount: 0, UploaderID: in.Actor.UserID, URL: url, CreatedAt: time.Now().Unix(),
	}}, nil
}

func validateUpload(in *filepb.UploadFileRequest) (name, ext, mime, scene, visibility string, err error) {
	invalid := func(msg string) (string, string, string, string, string, error) {
		return "", "", "", "", "", xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, msg)
	}
	if len(in.Content) == 0 || len(in.Content) > 10<<20 {
		return invalid("文件大小须为 1 字节至 10 MiB")
	}
	name = strings.TrimSpace(in.OriginalName)
	if name == "" || !utf8.ValidString(name) || len(name) > 255 || strings.ContainsAny(name, `/\:`) || strings.HasSuffix(name, ".") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return invalid("文件名不合法")
	}
	scene = in.Scene
	if scene == "" {
		scene = "common"
	}
	if scene != "common" && scene != "avatar" && scene != "doc" {
		return invalid("文件场景不合法")
	}
	visibility = in.Visibility
	if visibility == "" {
		visibility = "private"
	}
	if visibility != "private" && visibility != "public" {
		return invalid("文件可见性不合法")
	}
	ext = strings.ToLower(filepath.Ext(name))
	mime = strings.Split(http.DetectContentType(in.Content), ";")[0]
	imageExts := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif"}
	if expected, ok := imageExts[ext]; ok {
		if mime != expected {
			return invalid("图片内容与扩展名不匹配")
		}
		cfg, _, decodeErr := image.DecodeConfig(bytes.NewReader(in.Content))
		if decodeErr != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 4096 || cfg.Height > 4096 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 {
			return invalid("图片损坏或尺寸超过限制")
		}
		if _, _, decodeErr = image.Decode(bytes.NewReader(in.Content)); decodeErr != nil {
			return invalid("图片无法解码")
		}
	} else {
		if scene == "avatar" {
			return invalid("头像仅支持 JPEG、PNG、GIF")
		}
		switch ext {
		case ".pdf":
			if mime != "application/pdf" {
				return invalid("PDF 内容与扩展名不匹配")
			}
		case ".txt", ".csv":
			prefix := strings.ToLower(string(in.Content[:min(len(in.Content), 512)]))
			if mime != "text/plain" || !utf8.Valid(in.Content) || strings.Contains(prefix, "<svg") || strings.Contains(prefix, "<html") || strings.Contains(prefix, "<script") {
				return invalid("文本内容不合法")
			}
			if ext == ".csv" {
				mime = "text/csv"
			}
		default:
			return invalid("支持 JPEG、PNG、GIF、PDF、TXT、CSV 文件")
		}
	}
	return name, ext, mime, scene, visibility, nil
}

func fileModelError(err error) error {
	switch {
	case errors.Is(err, model.ErrNotFound):
		return xerr.NewErrCodeMsg(xerr.NOT_FOUND, "文件不存在或已删除")
	case errors.Is(err, model.ErrFileForbidden):
		return xerr.NewErrCode(xerr.PERMISSION_DENIED)
	case errors.Is(err, model.ErrFileReferenced):
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件仍被引用，无法删除")
	case errors.Is(err, model.ErrFileNotAvatar):
		return xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "请选择通过头像场景上传的图片")
	default:
		return xerr.NewErrCode(xerr.DB_ERROR)
	}
}

// buildObjectKey 生成对象存储键：scene/yyyy/mm/uuid.ext
func buildObjectKey(scene, ext string) string {
	if scene == "" {
		scene = "common"
	}
	now := time.Now()
	return fmt.Sprintf("%s/%04d/%02d/%s%s",
		scene, now.Year(), int(now.Month()), uuid.NewString(), ext)
}
