package fileresourceservicelogic

import (
	"bytes"
	"context"
	"net/http"
	"path"
	"strings"
	"unicode/utf8"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/pkg/result/xerr"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/gofrs/uuid/v5"
)

const (
	maxUploadBytes     = 10 << 20 // 10MB hard cap, matches the API tier
	privateURLLifetime = 300      // seconds a private OSS signed URL stays valid
)

// ossBucket returns the configured bucket or an error if OSS is not configured.
func ossBucket(svcCtx *svc.ServiceContext) (*oss.Bucket, error) {
	if svcCtx.OssClient == nil {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务未配置")
	}
	cfg := svcCtx.Config.Oss
	if cfg.BucketName == "" {
		return nil, xerr.NewErrCodeMsg(300002, "文件存储服务配置无效")
	}
	return svcCtx.OssClient.Bucket(cfg.BucketName)
}

// imageExtension maps the detected MIME to the canonical object key extension.
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

// normalizeUploadFilename strips directory components and rejects dangerous
// inputs. This is duplicated from the API tier to keep both layers defensive
// even if one is bypassed.
func normalizeUploadFilename(raw string) (string, error) {
	name := path.Base(strings.ReplaceAll(raw, "\\", "/"))
	if strings.TrimSpace(name) == "" || name == "." || name == ".." || len([]rune(name)) > 255 || !utf8.ValidString(name) || strings.ContainsAny(name, "\x00\r\n") {
		return "", xerr.NewErrCodeMsg(300002, "文件名无效或超过255字符")
	}
	return name, nil
}

// newImageKey produces a uuid-based object key under the project's prefix.
func newImageKey(mime string) (string, error) {
	id, err := uuid.NewV4()
	if err != nil {
		return "", err
	}
	return "go-zero-admin/" + id.String() + imageExtension(mime), nil
}

// detectAndValidateImage checks size and MIME for the upload. We re-detect the
// MIME from the bytes rather than trusting the client-supplied value.
func detectAndValidateImage(data []byte) (string, error) {
	if len(data) == 0 || len(data) > maxUploadBytes {
		return "", xerr.NewErrCodeMsg(300002, "图片不能为空且不能超过10MB")
	}
	mime := http.DetectContentType(data)
	if imageExtension(mime) == "" {
		return "", xerr.NewErrCodeMsg(300002, "仅支持PNG、JPEG、GIF、WebP图片")
	}
	return mime, nil
}

// publicFileURL builds the canonical public URL for a public object.
func publicFileURL(bucketName, endpoint, key string) string {
	endpoint = strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(endpoint, "https://"), "http://"), "/")
	return "https://" + bucketName + "." + endpoint + "/" + key
}

// putObject writes the bytes to OSS with the requested ACL and content type.
func putObject(ctx context.Context, bucket *oss.Bucket, key, mime string, data []byte, visibility string) error {
	acl := oss.ACLPublicRead
	if visibility == "private" {
		acl = oss.ACLPrivate
	}
	return bucket.PutObject(key, bytes.NewReader(data), oss.ContentType(mime), oss.ObjectACL(acl), oss.WithContext(ctx))
}
