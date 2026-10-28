package fileresourceservicelogic

import (
	"context"
	"errors"
	"go-zero-admin/pkg/testmysql"
	"sync"
	"testing"
	"time"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

// This integration test requires CREATE/DROP DATABASE privileges and always uses
// its own fresh database. The configured DSN's database is never opened or changed.
func mysqlFileFixture(t *testing.T) (*svc.ServiceContext, *pb.SessionRequest, int64) {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&model.SysAuthority{}, &model.SysUser{}, &model.SysUserAuthority{}, &model.SysFileResource{}, &model.SysFileReference{}, &model.SysAuditLog{}); err != nil {
		t.Fatal(err)
	}
	for _, value := range []any{
		&model.SysAuthority{AuthorityId: 1, AuthorityName: "admin"},
		&model.SysUser{MODEL_BASE: base.MODEL_BASE{ID: 30}, Username: "test-admin", AuthorityId: 1, Enable: 1, SessionVersion: 1},
		&model.SysUserAuthority{SysUserId: 30, SysAuthorityAuthorityId: 1},
	} {
		if err := db.Omit("Authority", "Authorities").Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	file := model.SysFileResource{ObjectKey: "go-zero-admin/10000000-0000-4000-8000-000000000001.png", Name: "referenced.png", Mime: "image/png", Size: 100, OwnerID: 30, Visibility: "private", Status: "active"}
	if err := db.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	return &svc.ServiceContext{DB: &orm.DB{DB: db}}, &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1}, file.ID
}

type mysqlFileTransactionKey struct{}

func TestMySQLRepeatableReadReferenceCommittedWhileDeleteWaits(t *testing.T) {
	s, actor, fileID := mysqlFileFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	referenceLocked, deletionWaiting, releaseReference := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce, deletionOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseReference) }) }
	defer release()
	if err := s.DB.Callback().Create().Before("gorm:create").Register("test:hold-file-reference", func(tx *gorm.DB) {
		if tx.Statement.Table != "sys_file_references" || tx.Statement.Context.Value(mysqlFileTransactionKey{}) != "reference" {
			return
		}
		// AddFileReference already owns the resource row lock. Pause before its
		// insert so deletion's session lookup establishes a reference-free snapshot.
		close(referenceLocked)
		select {
		case <-releaseReference:
		case <-ctx.Done():
			tx.AddError(ctx.Err())
		}
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Callback().Query().Before("gorm:query").Register("test:observe-delete-file-lock", func(tx *gorm.DB) {
		if tx.Statement.Table == "sys_file_resources" && tx.Statement.Context.Value(mysqlFileTransactionKey{}) == "delete" {
			// datascope.Apply's ordinary session SELECT has finished. The next
			// SELECT FOR UPDATE waits for the reference transaction's resource lock.
			deletionOnce.Do(func() { close(deletionWaiting) })
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = s.DB.Callback().Create().Remove("test:hold-file-reference")
		_ = s.DB.Callback().Query().Remove("test:observe-delete-file-lock")
	})
	referenceDone, deletionDone := make(chan error, 1), make(chan error, 1)
	go func() {
		_, err := NewAddFileReferenceLogic(context.WithValue(ctx, mysqlFileTransactionKey{}, "reference"), s).AddFileReference(&pb.FileReferenceRequest{Actor: actor, FileID: fileID, ObjectType: "article", ObjectID: "7"})
		referenceDone <- err
	}()
	wait := func(signal <-chan struct{}, phase string) {
		t.Helper()
		select {
		case <-signal:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s", phase)
		}
	}
	wait(referenceLocked, "reference holding the file row")
	go func() {
		_, err := NewBeginDeleteFileLogic(context.WithValue(ctx, mysqlFileTransactionKey{}, "delete"), s).BeginDeleteFile(&pb.FileIDRequest{Actor: actor, ID: fileID})
		deletionDone <- err
	}()
	wait(deletionWaiting, "deletion after its snapshot read")
	release()
	if err := <-referenceDone; err != nil {
		t.Fatal("reference registration failed: ", err)
	}
	deleteErr := <-deletionDone
	if deleteErr == nil {
		t.Fatal("deletion bypassed a committed reference because it read the old REPEATABLE READ snapshot")
	}
	var rejection *xerr.CodeError
	if !errors.As(deleteErr, &rejection) || rejection.GetErrCode() != xerr.REUQEST_PARAM_ERROR || rejection.GetErrMsg() != "文件仍被业务引用，不能删除" {
		t.Fatalf("deletion failed for an unexpected reason: %v", deleteErr)
	}
	var file model.SysFileResource
	if err := s.DB.First(&file, fileID).Error; err != nil {
		t.Fatal(err)
	}
	if file.Status != "active" {
		t.Fatalf("referenced file changed to %s", file.Status)
	}
	var references int64
	if err := s.DB.Model(&model.SysFileReference{}).Where("file_id = ?", fileID).Count(&references).Error; err != nil || references != 1 {
		t.Fatalf("reference was lost: count=%d err=%v", references, err)
	}
}
