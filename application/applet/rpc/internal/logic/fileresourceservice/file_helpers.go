package fileresourceservicelogic

import (
	"context"
	"errors"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/gofrs/uuid/v5"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func fileError(message string) error { return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, message) }

func scopedFile(ctx context.Context, db *gorm.DB, actor *pb.SessionRequest, id int64, lock bool) (*model.SysFileResource, error) {
	if id <= 0 {
		return nil, fileError("文件ID无效")
	}
	query, err := datascope.Apply(ctx, db.Model(&model.SysFileResource{}), actor, "owner_id", "department_id")
	if err != nil {
		return nil, err
	}
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var file model.SysFileResource
	if err := query.Where("id = ?", id).First(&file).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fileError("文件不存在或不在可访问范围")
		}
		return nil, err
	}
	return &file, nil
}

func fileProto(ctx context.Context, db *gorm.DB, file *model.SysFileResource) (*pb.FileResource, error) {
	var refs int64
	if err := db.WithContext(ctx).Model(&model.SysFileReference{}).Where("file_id = ?", file.ID).Count(&refs).Error; err != nil {
		return nil, err
	}
	return fileProtoWithReferences(file, refs), nil
}

// The caller must already own the resource row lock, as reference mutations do.
// A plain COUNT can use the session lookup's older REPEATABLE READ snapshot even
// after the resource lock becomes available. This locking read sees references
// that committed while we were waiting, and needs only one row to reject deletion.
func hasLockedFileReferences(ctx context.Context, tx *gorm.DB, fileID int64) (bool, error) {
	var reference model.SysFileReference
	err := tx.WithContext(ctx).Select("id").Where("file_id = ?", fileID).
		Clauses(clause.Locking{Strength: "UPDATE"}).Take(&reference).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return err == nil, err
}

func fileProtoWithReferences(file *model.SysFileResource, refs int64) *pb.FileResource {
	return &pb.FileResource{ID: file.ID, ObjectKey: file.ObjectKey, Name: file.Name, Mime: file.Mime, Size: file.Size,
		OwnerID: file.OwnerID, DepartmentId: file.DepartmentID, Visibility: file.Visibility, Status: file.Status,
		References: refs, CreatedAt: file.CreatedAt.UTC().Format(time.RFC3339Nano)}
}

func auditFile(ctx context.Context, tx *gorm.DB, actor *pb.SessionRequest, action string, id int64) error {
	identity := audit.ActorFromContext(ctx)
	if identity.Name == "" {
		var user model.SysUser
		if err := tx.Select("username").First(&user, "id = ?", actor.UserID).Error; err != nil {
			return err
		}
		identity.Name = user.Username
	}
	return audit.Record(ctx, tx, audit.Event{ActorID: actor.UserID, ActorName: identity.Name, AuthorityID: actor.AuthorityId,
		Module: "files", Action: action, Object: strconv.FormatInt(id, 10)})
}

func validateFile(input *pb.FileResource) error {
	if input == nil || input.Size <= 0 || input.Size > 10<<20 || len([]rune(input.Name)) > 255 || !utf8.ValidString(input.Name) || strings.TrimSpace(input.Name) == "" || strings.ContainsAny(input.Name, "\x00\r\n") {
		return fileError("文件元数据无效")
	}
	if input.Visibility != "public" && input.Visibility != "private" {
		return fileError("文件可见性只允许public或private")
	}
	extensions := map[string]string{"image/png": ".png", "image/jpeg": ".jpg", "image/gif": ".gif", "image/webp": ".webp"}
	ext := extensions[input.Mime]
	if ext == "" || !strings.HasPrefix(input.ObjectKey, "go-zero-admin/") || path.Ext(input.ObjectKey) != ext {
		return fileError("只支持系统上传的PNG、JPEG、GIF、WebP资源")
	}
	id := strings.TrimSuffix(strings.TrimPrefix(input.ObjectKey, "go-zero-admin/"), ext)
	if _, err := uuid.FromString(id); err != nil {
		return fileError("文件对象标识无效")
	}
	return nil
}

var objectTypePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_-]{0,63}$`)
var objectIDPattern = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_.:-]{0,127}$`)

func validateReference(input *pb.FileReferenceRequest) error {
	if input == nil || input.Actor == nil || input.Actor.AuthorityId != 1 {
		return fileError("首版仅内置管理员可管理业务引用")
	}
	if input.FileID <= 0 || !objectTypePattern.MatchString(input.ObjectType) || !objectIDPattern.MatchString(input.ObjectID) {
		return fileError("文件引用标识无效")
	}
	return nil
}
