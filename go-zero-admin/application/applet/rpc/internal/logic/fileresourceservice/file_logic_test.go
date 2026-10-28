package fileresourceservicelogic

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func fileDB(t *testing.T) (*svc.ServiceContext, *pb.SessionRequest, *pb.SessionRequest) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysAuthority{}, &model.SysUserAuthority{}, &model.SysDepartment{}, &model.SysUserDepartment{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &model.SysDeviceSession{}, &model.SysFileResource{}, &model.SysFileReference{}, &model.SysAuditLog{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if err := db.Create(&model.SysAuthority{AuthorityId: id, AuthorityName: "role"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, user := range []model.SysUser{{MODEL_BASE: base.MODEL_BASE{ID: 10}, Username: "member", AuthorityId: 2, Enable: 1, SessionVersion: 1}, {MODEL_BASE: base.MODEL_BASE{ID: 30}, Username: "admin", AuthorityId: 1, Enable: 1, SessionVersion: 1}} {
		if err := db.Omit("Authority", "Authorities").Create(&user).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.SysUserAuthority{SysUserId: user.ID, SysAuthorityAuthorityId: user.AuthorityId}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Create(&model.SysDepartment{MODEL_BASE: base.MODEL_BASE{ID: 2}, Name: "dept", Code: "dept", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&model.SysUserDepartment{UserID: 10, DepartmentID: 2}).Error; err != nil {
		t.Fatal(err)
	}
	return &svc.ServiceContext{DB: &orm.DB{DB: db}}, &pb.SessionRequest{UserID: 10, AuthorityId: 2, SessionVersion: 1}, &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1}
}
func fileInput(actor *pb.SessionRequest, number string) *pb.RegisterFileRequest {
	return &pb.RegisterFileRequest{Actor: actor, File: &pb.FileResource{ObjectKey: "go-zero-admin/10000000-0000-4000-8000-00000000000" + number + ".png", Name: "example.png", Mime: "image/png", Size: 100, Visibility: "private", OwnerID: 999, DepartmentId: 999}}
}

func TestFilesOwnershipDataScopeAndIdempotentRegister(t *testing.T) {
	s, member, admin := fileDB(t)
	ctx := context.Background()
	input := fileInput(member, "1")
	file, err := NewRegisterFileLogic(ctx, s).RegisterFile(input)
	if err != nil {
		t.Fatal(err)
	}
	if file.OwnerID != member.UserID || file.DepartmentId != 2 || file.References != 0 {
		t.Fatalf("trusted caller metadata: %+v", file)
	}
	retry, err := NewRegisterFileLogic(ctx, s).RegisterFile(input)
	if err != nil || retry.ID != file.ID {
		t.Fatalf("register retry not idempotent: %+v err=%v", retry, err)
	}
	other, err := NewRegisterFileLogic(ctx, s).RegisterFile(fileInput(admin, "2"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := NewGetFileLogic(ctx, s).GetFile(&pb.FileIDRequest{Actor: member, ID: other.ID}); err == nil {
		t.Fatal("member read another user's file")
	}
	if _, err := NewBeginDeleteFileLogic(ctx, s).BeginDeleteFile(&pb.FileIDRequest{Actor: member, ID: other.ID}); err == nil {
		t.Fatal("member deleted another user's file")
	}
	response, err := NewGetFileListLogic(ctx, s).GetFileList(&pb.FileListRequest{Actor: member, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}})
	if err != nil || response.Total != 1 || response.List[0].ID != file.ID {
		t.Fatalf("data scope not applied to list: %+v err=%v", response, err)
	}
	response, err = NewGetFileListLogic(ctx, s).GetFileList(&pb.FileListRequest{Actor: admin, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}})
	if err != nil || response.Total != 2 {
		t.Fatalf("admin list restricted: %+v err=%v", response, err)
	}
	var logs int64
	if err := s.DB.Model(&model.SysAuditLog{}).Count(&logs).Error; err != nil || logs != 2 {
		t.Fatalf("missing/duplicate transaction audit count=%d err=%v", logs, err)
	}
}

func TestFileReferencesProtectDeleteAndRetries(t *testing.T) {
	s, member, admin := fileDB(t)
	ctx := context.Background()
	file, err := NewRegisterFileLogic(ctx, s).RegisterFile(fileInput(member, "1"))
	if err != nil {
		t.Fatal(err)
	}
	reference := &pb.FileReferenceRequest{Actor: member, FileID: file.ID, ObjectType: "article", ObjectID: "7"}
	if _, err := NewAddFileReferenceLogic(ctx, s).AddFileReference(reference); err == nil {
		t.Fatal("nonadmin managed unverified business reference")
	}
	reference.Actor = admin
	for i := 0; i < 2; i++ {
		if _, err := NewAddFileReferenceLogic(ctx, s).AddFileReference(reference); err != nil {
			t.Fatal(err)
		}
	}
	request := &pb.FileIDRequest{Actor: member, ID: file.ID}
	if _, err := NewBeginDeleteFileLogic(ctx, s).BeginDeleteFile(request); err == nil {
		t.Fatal("referenced file entered deletion")
	}
	if _, err := NewFinishDeleteFileLogic(ctx, s).FinishDeleteFile(request); err == nil {
		t.Fatal("finish bypassed deletion phase")
	}
	if _, err := NewRemoveFileReferenceLogic(ctx, s).RemoveFileReference(reference); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		deleting, err := NewBeginDeleteFileLogic(ctx, s).BeginDeleteFile(request)
		if err != nil || deleting.Status != "deleting" {
			t.Fatalf("delete retry invalid: %+v err=%v", deleting, err)
		}
	}
	if _, err := NewAddFileReferenceLogic(ctx, s).AddFileReference(reference); err == nil {
		t.Fatal("reference added while deleting")
	}
	if _, err := NewGetFileLogic(ctx, s).GetFile(request); err == nil {
		t.Fatal("deleting file still available")
	}
	for i := 0; i < 2; i++ {
		if _, err := NewFinishDeleteFileLogic(ctx, s).FinishDeleteFile(request); err != nil {
			t.Fatal(err)
		}
	}
	deleted, err := NewBeginDeleteFileLogic(ctx, s).BeginDeleteFile(request)
	if err != nil || deleted.Status != "deleted" {
		t.Fatalf("repeated completed delete invalid: %+v err=%v", deleted, err)
	}
	response, err := NewGetFileListLogic(ctx, s).GetFileList(&pb.FileListRequest{Actor: member, PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 10}})
	if err != nil || response.Total != 0 {
		t.Fatal("deleted file remains in default list")
	}
}

func TestFileAuditFailureRollsBackRegistrationAndMetadataValidation(t *testing.T) {
	s, member, _ := fileDB(t)
	ctx := context.Background()
	if err := s.DB.Exec("CREATE TRIGGER reject_file_audit BEFORE INSERT ON sys_audit_logs WHEN NEW.module = 'files' BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewRegisterFileLogic(ctx, s).RegisterFile(fileInput(member, "1")); err == nil {
		t.Fatal("audit failure did not fail registration")
	}
	var count int64
	if err := s.DB.Model(&model.SysFileResource{}).Count(&count).Error; err != nil || count != 0 {
		t.Fatal("registration not rolled back")
	}
	for _, modify := range []func(*pb.FileResource){func(f *pb.FileResource) { f.ObjectKey = "unrelated/secret" }, func(f *pb.FileResource) { f.Size = 11 << 20 }, func(f *pb.FileResource) { f.Visibility = "unknown" }, func(f *pb.FileResource) { f.Mime = "application/javascript" }} {
		input := fileInput(member, "1")
		modify(input.File)
		if err := validateFile(input.File); err == nil {
			t.Fatal("unsafe file metadata accepted")
		}
	}
}
