package fileservicelogic

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/zeromicro/go-zero/core/logx"

	filepb "go-zero-admin/application/file/rpc/filepb"
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
	if l.svcCtx.Store == nil {
		return nil, xerr.NewErrCodeMsg(500, "对象存储未配置")
	}
	// 校验大小（10MB 上限，与迁移注释一致）
	const maxSize = 10 << 20
	if len(in.Content) == 0 {
		return nil, xerr.NewErrCodeMsg(400, "文件内容为空")
	}
	if len(in.Content) > maxSize {
		return nil, xerr.NewErrCodeMsg(400, "文件超过 10MB 限制")
	}

	// 生成对象键：scene/yyyy/mm/uuid.ext
	ext := strings.ToLower(filepath.Ext(in.OriginalName))
	objectKey := buildObjectKey(in.Scene, ext)

	visibility := in.Visibility
	if visibility != "public" && visibility != "private" {
		visibility = "public"
	}

	if err := l.svcCtx.Store.Put(l.ctx, objectKey, in.Mime, in.Content, visibility); err != nil {
		l.Errorf("store put error: %v", err)
		return nil, xerr.NewErrCodeMsg(500, "文件上传失败")
	}

	bucket := l.svcCtx.Config.Storage.PublicBucket
	if visibility == "private" {
		bucket = l.svcCtx.Config.Storage.PrivateBucket
	}
	size := int64(len(in.Content))

	result, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO sys_file (bucket, object_key, original_name, ext, mime, size, visibility, uploader_id) VALUES (?,?,?,?,?,?,?,?)",
		bucket, objectKey, in.OriginalName, ext, in.Mime, size, visibility, in.Actor.UserID)
	if err != nil {
		l.Errorf("insert sys_file error: %v", err)
		// 补偿：删除已上传的对象
		_ = l.svcCtx.Store.Delete(l.ctx, objectKey, visibility)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	id, _ := result.LastInsertId()

	// public 直接给 URL，private 需调 GetFileURL 获取签名
	var url string
	if visibility == "public" {
		url, _, _ = l.svcCtx.Store.URL(l.ctx, objectKey, "public")
	}

	return &filepb.UploadFileResponse{File: &filepb.FileInfo{
		Id: id, Bucket: bucket, ObjectKey: objectKey, OriginalName: in.OriginalName,
		Ext: ext, Mime: in.Mime, Size: size, Visibility: visibility,
		RefCount: 0, UploaderID: in.Actor.UserID, URL: url,
	}}, nil
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
