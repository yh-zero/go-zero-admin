package fileservicelogic

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
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
	digest := sha256.Sum256(in.Content)
	return persistUpload(l.ctx, l.svcCtx, in.Actor, persistUploadInput{
		Name: name, Ext: ext, Mime: mime, Scene: scene, Visibility: visibility,
		Body: bytes.NewReader(in.Content), Size: int64(len(in.Content)), SHA256: hex.EncodeToString(digest[:]),
	})
}

type persistUploadInput struct {
	Name       string
	Ext        string
	Mime       string
	Scene      string
	Visibility string
	Body       io.Reader
	Size       int64
	SHA256     string
}

func persistUpload(ctx context.Context, svcCtx *svc.ServiceContext, actor *filepb.Actor, input persistUploadInput) (*filepb.UploadFileResponse, error) {
	if svcCtx.Store == nil {
		return nil, xerr.NewErrCodeMsg(xerr.SERVICE_UNAVAILABLE, "对象存储未配置")
	}
	objectKey := buildObjectKey(input.Scene, input.Ext)
	if err := svcCtx.Store.PutReader(ctx, objectKey, input.Mime, input.Body, input.Size, input.Visibility); err != nil {
		cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		_ = svcCtx.Store.Delete(cleanupCtx, objectKey, input.Visibility)
		return nil, xerr.NewErrCodeMsg(500, "文件上传失败")
	}
	bucket := svcCtx.Config.Storage.PublicBucket
	if input.Visibility == "private" {
		bucket = svcCtx.Config.Storage.PrivateBucket
	}
	result, err := svcCtx.SysFileModel.Insert(ctx, &model.SysFile{
		Bucket: bucket, ObjectKey: objectKey, OriginalName: input.Name, Ext: input.Ext, Mime: input.Mime,
		Size: uint64(input.Size), Sha256: input.SHA256, Visibility: input.Visibility,
		UploaderId: sql.NullInt64{Int64: actor.UserID, Valid: true},
	})
	if err != nil {
		if result == nil {
			cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
			defer cancel()
			_ = svcCtx.Store.Delete(cleanupCtx, objectKey, input.Visibility)
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	var url string
	if input.Visibility == "public" {
		url, _, err = svcCtx.Store.URL(ctx, objectKey, "public")
		if err != nil {
			return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
	}
	return &filepb.UploadFileResponse{File: &filepb.FileInfo{
		Id: id, Bucket: bucket, ObjectKey: objectKey, OriginalName: input.Name,
		Ext: input.Ext, Mime: input.Mime, Size: input.Size, Sha256: input.SHA256, Visibility: input.Visibility,
		RefCount: 0, UploaderID: actor.UserID, UploaderName: actor.Username, URL: url, CreatedAt: time.Now().Unix(),
	}}, nil
}

func validateUpload(in *filepb.UploadFileRequest) (name, ext, mime, scene, visibility string, err error) {
	if len(in.Content) == 0 || len(in.Content) > 10<<20 {
		return "", "", "", "", "", xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "文件大小须为 1 字节至 10 MiB")
	}
	name, ext, scene, visibility, err = validateUploadMetadata(in.OriginalName, in.Scene, in.Visibility)
	if err != nil {
		return "", "", "", "", "", err
	}
	mime, err = validateUploadContent(bytes.NewReader(in.Content), ext, scene)
	return name, ext, mime, scene, visibility, err
}

func validateUploadMetadata(originalName, scene, visibility string) (name, ext, normalizedScene, normalizedVisibility string, err error) {
	invalid := func(msg string) (string, string, string, string, error) {
		return "", "", "", "", xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, msg)
	}
	name = strings.TrimSpace(originalName)
	if name == "" || !utf8.ValidString(name) || len(name) > 255 || strings.ContainsAny(name, `/\:`) || strings.HasSuffix(name, ".") || strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return invalid("文件名不合法")
	}
	normalizedScene = scene
	if normalizedScene == "" {
		normalizedScene = "common"
	}
	if normalizedScene != "common" && normalizedScene != "avatar" && normalizedScene != "doc" {
		return invalid("文件场景不合法")
	}
	normalizedVisibility = visibility
	if normalizedVisibility == "" {
		normalizedVisibility = "private"
	}
	if normalizedVisibility != "private" && normalizedVisibility != "public" {
		return invalid("文件可见性不合法")
	}
	return name, strings.ToLower(filepath.Ext(name)), normalizedScene, normalizedVisibility, nil
}

func validateUploadContent(reader io.ReadSeeker, ext, scene string) (string, error) {
	prefix := make([]byte, 512)
	n, err := io.ReadFull(reader, prefix)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, "读取文件内容失败")
	}
	prefix = prefix[:n]
	mime := strings.Split(http.DetectContentType(prefix), ";")[0]
	invalid := func(msg string) (string, error) {
		return "", xerr.NewErrCodeMsg(xerr.REQUEST_PARAM_ERROR, msg)
	}
	imageExts := map[string]string{".jpg": "image/jpeg", ".jpeg": "image/jpeg", ".png": "image/png", ".gif": "image/gif"}
	if expected, ok := imageExts[ext]; ok {
		if mime != expected {
			return invalid("图片内容与扩展名不匹配")
		}
		if _, err = reader.Seek(0, io.SeekStart); err != nil {
			return invalid("读取图片失败")
		}
		cfg, _, decodeErr := image.DecodeConfig(reader)
		if decodeErr != nil || cfg.Width <= 0 || cfg.Height <= 0 || cfg.Width > 4096 || cfg.Height > 4096 || int64(cfg.Width)*int64(cfg.Height) > 16_000_000 {
			return invalid("图片损坏或尺寸超过限制")
		}
		if _, err = reader.Seek(0, io.SeekStart); err != nil {
			return invalid("读取图片失败")
		}
		if _, _, decodeErr = image.Decode(reader); decodeErr != nil {
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
			lowerPrefix := strings.ToLower(string(prefix))
			if mime != "text/plain" || strings.Contains(lowerPrefix, "<svg") || strings.Contains(lowerPrefix, "<html") || strings.Contains(lowerPrefix, "<script") {
				return invalid("文本内容不合法")
			}
			if _, err = reader.Seek(0, io.SeekStart); err != nil {
				return invalid("读取文本失败")
			}
			if !validUTF8Stream(reader) {
				return invalid("文本内容不合法")
			}
			if ext == ".csv" {
				mime = "text/csv"
			}
		default:
			return invalid("支持 JPEG、PNG、GIF、PDF、TXT、CSV 文件")
		}
	}
	return mime, nil
}

func validUTF8Stream(reader io.Reader) bool {
	buffered := bufio.NewReader(reader)
	for {
		r, size, err := buffered.ReadRune()
		if errors.Is(err, io.EOF) {
			return true
		}
		if err != nil || (r == utf8.RuneError && size == 1) {
			return false
		}
	}
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

func buildObjectKey(scene, ext string) string {
	if scene == "" {
		scene = "common"
	}
	now := time.Now()
	return fmt.Sprintf("%s/%04d/%02d/%s%s", scene, now.Year(), int(now.Month()), uuid.NewString(), ext)
}
