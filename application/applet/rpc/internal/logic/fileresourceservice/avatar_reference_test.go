package fileresourceservicelogic

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-zero-admin/application/applet/rpc/internal/logic/avatarutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/objectstorage"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func TestAutomaticAvatarReferencesCannotBeManuallyChanged(t *testing.T) {
	s, member, admin := fileDB(t)
	file, err := NewRegisterFileLogic(context.Background(), s).RegisterFile(fileInput(member, "1"))
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{avatarutil.SysUser, avatarutil.AppUser} {
		reference := model.SysFileReference{FileID: file.ID, ObjectType: kind, ObjectID: "10"}
		if err := s.DB.Create(&reference).Error; err != nil {
			t.Fatal(err)
		}
		input := &pb.FileReferenceRequest{Actor: admin, FileID: file.ID, ObjectType: kind, ObjectID: "10"}
		if _, err := NewAddFileReferenceLogic(context.Background(), s).AddFileReference(input); err == nil {
			t.Fatal("manual API accepted reserved avatar reference")
		}
		if _, err := NewRemoveFileReferenceLogic(context.Background(), s).RemoveFileReference(input); err == nil {
			t.Fatal("manual API removed reserved avatar reference")
		}
	}
	if _, err := NewBeginDeleteFileLogic(context.Background(), s).BeginDeleteFile(&pb.FileIDRequest{Actor: admin, ID: file.ID}); err == nil {
		t.Fatal("in-use avatar was deletable")
	}
	var count int64
	if err := s.DB.Model(&model.SysFileReference{}).Where("file_id = ?", file.ID).Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("system references changed: %d %v", count, err)
	}
}

func TestMySQLAvatarCommittedWhileFileDeletionWaits(t *testing.T) {
	s, actor, fileID := mysqlFileFixture(t)
	if err := s.DB.Model(&model.SysFileResource{}).Where("id = ?", fileID).Update("visibility", "public").Error; err != nil {
		t.Fatal(err)
	}
	storage := objectstorage.Config{ExternalEndpoint: "http://127.0.0.1:9000", PublicBucket: "gozero-public", PrivateBucket: "gozero-private"}
	avatar := storage.ExternalEndpoint + "/gozero-public/go-zero-admin/10000000-0000-4000-8000-000000000001.png"
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	locked, deletionWaiting, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	if err := s.DB.Callback().Create().Before("gorm:create").Register("test:avatar-held", func(tx *gorm.DB) {
		if tx.Statement.Table != "sys_file_references" {
			return
		}
		close(locked)
		select {
		case <-release:
		case <-ctx.Done():
			tx.AddError(ctx.Err())
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Callback().Query().Before("gorm:query").Register("test:avatar-delete-waits", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_file_resources" && tx.Statement.Context.Value(mysqlFileTransactionKey{}) == "avatar-delete" {
			close(deletionWaiting)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = s.DB.Callback().Create().Remove("test:avatar-held")
		_ = s.DB.Callback().Query().Remove("test:avatar-delete-waits")
	})
	avatarDone, deleteDone := make(chan error, 1), make(chan error, 1)
	go func() {
		avatarDone <- s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var user model.SysUser
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, actor.UserID).Error; err != nil {
				return err
			}
			if err := avatarutil.Sync(ctx, tx, storage, avatarutil.SysUser, user.ID, user.HeaderImg, avatar); err != nil {
				return err
			}
			return tx.Model(&user).Update("header_img", avatar).Error
		})
	}()
	select {
	case <-locked:
	case <-ctx.Done():
		t.Fatal("avatar did not lock file")
	}
	go func() {
		_, err := NewBeginDeleteFileLogic(context.WithValue(ctx, mysqlFileTransactionKey{}, "avatar-delete"), s).BeginDeleteFile(&pb.FileIDRequest{Actor: actor, ID: fileID})
		deleteDone <- err
	}()
	select {
	case <-deletionWaiting:
	case <-ctx.Done():
		t.Fatal("deletion did not wait for avatar")
	}
	close(release)
	if err := <-avatarDone; err != nil {
		t.Fatal(err)
	}
	deleteErr := <-deleteDone
	var rejection *xerr.CodeError
	if !errors.As(deleteErr, &rejection) || rejection.GetErrCode() != xerr.REUQEST_PARAM_ERROR || rejection.GetErrMsg() != "文件仍被业务引用，不能删除" {
		t.Fatalf("deletion must reject the committed avatar reference, got %v", deleteErr)
	}
	var file model.SysFileResource
	if err := s.DB.First(&file, fileID).Error; err != nil || file.Status != "active" {
		t.Fatalf("avatar file changed: %+v %v", file, err)
	}
	var user model.SysUser
	if err := s.DB.First(&user, actor.UserID).Error; err != nil || user.HeaderImg != avatar {
		t.Fatalf("avatar update was lost: %+v %v", user, err)
	}
	var references int64
	if err := s.DB.Model(&model.SysFileReference{}).Where("file_id = ? AND object_type = ? AND object_id = ?", fileID, avatarutil.SysUser, "30").Count(&references).Error; err != nil || references != 1 {
		t.Fatalf("avatar reference was lost: %d %v", references, err)
	}
}
