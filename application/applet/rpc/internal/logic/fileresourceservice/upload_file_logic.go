package fileresourceservicelogic

import (
	"context"
	"path"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type UploadFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUploadFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UploadFileLogic {
	return &UploadFileLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

// UploadFile is the OSS-credentialed, one-shot upload used by the API tier.
// It validates the actor session against the database (a revoked session must
// not upload), validates the bytes, writes to OSS, and inserts the metadata
// row in a single transaction. On metadata failure the just-written blob is
// best-effort deleted: the key is a fresh UUID, so the only ambiguous case is
// a commit-time error, which is vanishingly rare and recoverable by re-upload.
func (l *UploadFileLogic) UploadFile(in *pb.UploadFileRequest) (*pb.UploadFileResponse, error) {
	if in == nil || in.Actor == nil {
		return nil, xerr.NewErrCodeMsg(300002, "上传请求无效")
	}
	// Validate the session is still live (session_version match, device not
	// revoked). RPC must not trust the API tier's middleware alone.
	if _, err := datascope.Apply(l.ctx, l.svcCtx.DB.Model(&model.SysFileResource{}), in.Actor, "owner_id", "department_id"); err != nil {
		return nil, err
	}
	if in.Visibility != "public" && in.Visibility != "private" {
		return nil, xerr.NewErrCodeMsg(300002, "文件可见性只允许public或private")
	}
	name, err := normalizeUploadFilename(in.Filename)
	if err != nil {
		return nil, err
	}
	mime, err := detectAndValidateImage(in.Data)
	if err != nil {
		return nil, err
	}
	if in.Mime != "" && in.Mime != mime {
		return nil, xerr.NewErrCodeMsg(300002, "文件类型与实际内容不一致")
	}
	key, err := newImageKey(mime)
	if err != nil {
		return nil, err
	}
	bucket, err := ossBucket(l.svcCtx)
	if err != nil {
		return nil, err
	}
	if err = putObject(l.ctx, bucket, key, mime, in.Data, in.Visibility); err != nil {
		l.Error("OSS upload failed")
		return nil, xerr.NewErrCodeMsg(300002, "图片上传失败，请检查文件存储服务")
	}

	// Inline the same idempotent metadata registration as RegisterFile. The
	// admin-role row lock serializes concurrent registrations of the same key
	// across RPC instances.
	file := model.SysFileResource{ObjectKey: key, Name: path.Base(strings.ReplaceAll(name, "\\", "/")), Mime: mime,
		Size: int64(len(in.Data)), OwnerID: in.Actor.UserID, Visibility: in.Visibility, Status: "active"}
	err = l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		var existing model.SysFileResource
		if err := tx.Where("object_key = ?", file.ObjectKey).First(&existing).Error; err == nil {
			if existing.OwnerID != file.OwnerID || existing.Name != file.Name || existing.Mime != file.Mime || existing.Size != file.Size || existing.Visibility != file.Visibility || existing.Status != "active" {
				return fileError("文件对象已登记且元数据不一致")
			}
			file = existing
			return nil
		}
		var membership model.SysUserDepartment
		if err := tx.Where("user_id = ?", in.Actor.UserID).First(&membership).Error; err == nil {
			file.DepartmentID = membership.DepartmentID
		}
		if err := tx.Create(&file).Error; err != nil {
			return err
		}
		return auditFile(l.ctx, tx, in.Actor, "registerFile", file.ID)
	})
	if err != nil {
		// Best-effort cleanup of the orphaned blob. A transaction error here
		// almost always means the row was rolled back; in the rare ambiguous
		// commit-failure case the user can simply re-upload.
		if delErr := bucket.DeleteObject(key, oss.WithContext(context.WithoutCancel(l.ctx))); delErr != nil {
			l.Errorf("orphan OSS object requires cleanup key=%s: %v", key, delErr)
		}
		return nil, err
	}
	resp := &pb.UploadFileResponse{ExpiresIn: 0}
	resp.File, err = fileProto(l.ctx, l.svcCtx.DB.DB, &file)
	if err != nil {
		return nil, err
	}
	cfg := l.svcCtx.Config.Oss
	if in.Visibility == "public" {
		resp.Url = publicFileURL(cfg.BucketName, cfg.Endpoint, key)
	} else {
		link, err := bucket.SignURL(key, oss.HTTPGet, privateURLLifetime)
		if err != nil {
			return nil, xerr.NewErrCodeMsg(300002, "图片已保存，访问地址生成失败")
		}
		resp.Url = link
		resp.ExpiresIn = privateURLLifetime
	}
	return resp, nil
}
