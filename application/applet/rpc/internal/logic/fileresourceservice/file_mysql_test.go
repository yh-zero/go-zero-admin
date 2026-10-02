package fileresourceservicelogic

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid/v5"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// This integration test requires CREATE/DROP DATABASE privileges and always uses
// its own fresh database. The configured DSN's database is never opened or changed.
func mysqlFileFixture(t *testing.T) (*svc.ServiceContext, *pb.SessionRequest, int64) {
	t.Helper()
	dsn := os.Getenv("GO_ZERO_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set GO_ZERO_MYSQL_TEST_DSN to run the isolated MySQL concurrency regression")
	}
	config, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid GO_ZERO_MYSQL_TEST_DSN")
	}
	config.DBName = ""
	config.Timeout, config.ReadTimeout, config.WriteTimeout = 5*time.Second, 15*time.Second, 15*time.Second
	admin, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal("cannot open MySQL test administration connection")
	}
	admin.SetMaxOpenConns(1)
	id, err := uuid.NewV4()
	if err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	const prefix = "gozero_file_rr_"
	name := prefix + strings.ReplaceAll(id.String(), "-", "")
	if !strings.HasPrefix(name, prefix) || len(name) != len(prefix)+32 || strings.ContainsAny(name, "`\\/ \x00") {
		_ = admin.Close()
		t.Fatal("invalid isolated test database identifier")
	}
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer setupCancel()
	if _, err = admin.ExecContext(setupCtx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4"); err != nil {
		_ = admin.Close()
		t.Fatal("cannot create isolated MySQL test database: ", err)
	}
	var database *sql.DB
	t.Cleanup(func() {
		if database != nil {
			_ = database.Close()
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(cleanupCtx, "DROP DATABASE `"+name+"`"); err != nil {
			t.Errorf("cannot remove isolated test database %s: %v", name, err)
		}
		_ = admin.Close()
	})
	config.DBName, config.ParseTime = name, true
	if config.Params == nil {
		config.Params = map[string]string{}
	}
	// Explicitly exercise the default InnoDB isolation, regardless of host settings.
	config.Params["transaction_isolation"] = "'REPEATABLE-READ'"
	db, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal("cannot open isolated MySQL test database: ", err)
	}
	database, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(4)
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
