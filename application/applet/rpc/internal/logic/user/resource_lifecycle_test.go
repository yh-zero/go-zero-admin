package userlogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/agentjobs"
	"go-zero-admin/pkg/audit"
	"testing"
	"time"
)

func lifecycleDB(t *testing.T) (*svc.ServiceContext, *pb.SessionRequest, *model.SysUser, *model.SysUser) {
	t.Helper()
	s := testUserDB(t)
	if err := s.DB.AutoMigrate(&model.SysFileResource{}, &model.SysFileReference{}, &model.SysDepartment{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &agentjobs.Conversation{}, &agentjobs.Message{}, &agentjobs.Run{}); err != nil {
		t.Fatal(err)
	}
	source := sessionUser(t, s, "source", 801)
	target := sessionUser(t, s, "target", 802)
	if err := s.DB.Create(&model.SysRoleDataScope{AuthorityID: 801, Scope: "all"}).Error; err != nil {
		t.Fatal(err)
	}
	file := model.SysFileResource{ObjectKey: "test/file", Name: "file", OwnerID: source.ID, Status: "active", Size: 100, Mime: "image/png"}
	if err := s.DB.Create(&file).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Create(&model.SysFileReference{FileID: file.ID, ObjectType: "article", ObjectID: "1"}).Error; err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	for _, value := range []any{&agentjobs.Conversation{ID: "conversation", OwnerID: source.ID, Title: "history", CreatedAt: now, UpdatedAt: now}, &agentjobs.Run{ID: "run", OwnerID: source.ID, AuthorityID: 801, ConversationID: "conversation", Sequence: 1, RequestID: "request", Status: agentjobs.StatusSucceeded, CreatedAt: now, UpdatedAt: now, QueueExpiresAt: now}, &agentjobs.Message{ID: "message", OwnerID: source.ID, ConversationID: "conversation", RunID: "run", Role: "assistant", Content: "retained history", CreatedAt: now}} {
		if err := s.DB.Create(value).Error; err != nil {
			t.Fatal(err)
		}
	}
	s.Config.DB.DataSource = "test@tcp(localhost:3306)/lifecycle"
	s.Config.UserResources.AIDataSource = s.Config.DB.DataSource
	return s, &pb.SessionRequest{UserID: source.ID, AuthorityId: 801, SessionVersion: source.SessionVersion}, source, target
}

func TestResourcePreviewAndAtomicTransfer(t *testing.T) {
	s, actor, source, target := lifecycleDB(t)
	ctx := context.Background()
	preview, err := NewGetUserResourcePreviewLogic(ctx, s).GetUserResourcePreview(&pb.UserResourcePreviewRequest{Actor: actor, UserID: source.ID})
	if err != nil || preview.Files != 1 || preview.AIConversations != 1 || preview.AIMessages != 1 || preview.AIRuns != 1 || !preview.AIAvailable || preview.TransferMode != "same_database" {
		t.Fatalf("resource preview lost ownership: %+v %v", preview, err)
	}
	moved, err := NewTransferUserResourcesLogic(ctx, s).TransferUserResources(&pb.TransferUserResourcesRequest{Actor: actor, UserID: source.ID, TargetUserID: target.ID, IncludeFiles: true, IncludeAI: true})
	if err != nil || moved.Files != 1 || moved.AIConversations != 1 || moved.AIMessages != 1 || moved.AIRuns != 1 || moved.TransactionMode != "same_database" {
		t.Fatalf("transaction result incorrect: %+v %v", moved, err)
	}
	for _, table := range []string{"sys_file_resources", "sys_ai_conversations", "sys_ai_messages", "sys_ai_runs"} {
		var count int64
		s.DB.Table(table).Where("owner_id = ?", target.ID).Count(&count)
		if count != 1 {
			t.Fatalf("%s was not transferred", table)
		}
	}
	var references int64
	s.DB.Model(&model.SysFileReference{}).Count(&references)
	if references != 1 {
		t.Fatal("transfer removed references")
	}
	var event audit.Event
	if err := s.DB.Where("action = ?", "transferUserResources").First(&event).Error; err != nil || event.ActorID != actor.UserID {
		t.Fatal("transfer missing actor audit", err)
	}
}

func TestResourceTransferRejectsScopeFrozenTargetsAndActiveRuns(t *testing.T) {
	for _, reason := range []string{"scope", "frozen", "deleted", "active", "different-database", "unconfigured-ai"} {
		t.Run(reason, func(t *testing.T) {
			s, actor, source, target := lifecycleDB(t)
			switch reason {
			case "scope":
				s.DB.Model(&model.SysRoleDataScope{}).Where("authority_id = ?", 801).Update("scope", "self")
			case "frozen":
				s.DB.Model(target).Update("enable", 2)
			case "deleted":
				s.DB.Delete(target)
			case "active":
				s.DB.Model(&agentjobs.Run{}).Where("id = ?", "run").Update("status", agentjobs.StatusQueued)
			case "different-database":
				s.Config.UserResources.AIDataSource = "test@tcp(other:3306)/lifecycle"
			case "unconfigured-ai":
				s.Config.UserResources.AIDataSource = ""
			}
			if _, err := NewTransferUserResourcesLogic(context.Background(), s).TransferUserResources(&pb.TransferUserResourcesRequest{Actor: actor, UserID: source.ID, TargetUserID: target.ID, IncludeFiles: true, IncludeAI: true}); err == nil {
				t.Fatal("invalid transfer accepted")
			}
			var count int64
			s.DB.Model(&model.SysFileResource{}).Where("owner_id = ?", source.ID).Count(&count)
			if count != 1 {
				t.Fatal("rejected transfer partially moved files")
			}
		})
	}
}

func TestResourceTransferAuditFailureRollsBackAndDeleteKeepsHistory(t *testing.T) {
	s, actor, source, target := lifecycleDB(t)
	if err := s.DB.Exec("CREATE TRIGGER reject_resource_audit BEFORE INSERT ON sys_audit_logs WHEN NEW.action = 'transferUserResources' BEGIN SELECT RAISE(FAIL, 'audit unavailable'); END").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewTransferUserResourcesLogic(context.Background(), s).TransferUserResources(&pb.TransferUserResourcesRequest{Actor: actor, UserID: source.ID, TargetUserID: target.ID, IncludeFiles: true, IncludeAI: true}); err == nil {
		t.Fatal("expected transaction audit failure")
	}
	if _, err := NewDeleteUserLogic(context.Background(), s).DeleteUser(&pb.DeleteUserRequest{UserID: source.ID}); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"sys_file_resources", "sys_ai_conversations", "sys_ai_messages", "sys_ai_runs"} {
		var count int64
		s.DB.Table(table).Where("owner_id = ?", source.ID).Count(&count)
		if count != 1 {
			t.Fatalf("delete/failed transfer removed or changed %s", table)
		}
	}
	var retained model.SysUser
	if err := s.DB.Unscoped().First(&retained, source.ID).Error; err != nil || !retained.DeletedAt.Valid {
		t.Fatal("user audit identity must remain soft deleted", err)
	}
}

func TestResourcePreviewReportsUnavailableAIWithoutClaimingZeroHistory(t *testing.T) {
	s, actor, source, _ := lifecycleDB(t)
	s.Config.UserResources.AIDataSource = "test@tcp(other:3306)/remote"
	result, err := NewGetUserResourcePreviewLogic(context.Background(), s).GetUserResourcePreview(&pb.UserResourcePreviewRequest{Actor: actor, UserID: source.ID})
	if err != nil || result.AIAvailable || result.AIReason == "" || result.Files != 1 || result.TransferMode != "files_only" {
		t.Fatalf("cross database AI must be explicit: %+v %v", result, err)
	}
}

func TestResourceTransferRequiresUsableTargetRoleAndDepartment(t *testing.T) {
	for _, reason := range []string{"role", "department"} {
		t.Run(reason, func(t *testing.T) {
			s, actor, source, target := lifecycleDB(t)
			if reason == "role" {
				s.DB.Where("sys_user_id = ?", target.ID).Delete(&model.SysUserAuthority{})
			} else {
				s.DB.Create(&model.SysUserDepartment{UserID: target.ID, DepartmentID: 99999})
			}
			if _, err := NewTransferUserResourcesLogic(context.Background(), s).TransferUserResources(&pb.TransferUserResourcesRequest{Actor: actor, UserID: source.ID, TargetUserID: target.ID, IncludeFiles: true}); err == nil {
				t.Fatal("target cannot access transferred resources because assignment is invalid")
			}
			var count int64
			s.DB.Model(&model.SysFileResource{}).Where("owner_id = ?", source.ID).Count(&count)
			if count != 1 {
				t.Fatal("invalid target received files")
			}
		})
	}
}

func TestResourceTransferAIUniqueCollisionRollsBackFilesAndHistory(t *testing.T) {
	s, actor, source, target := lifecycleDB(t)
	if err := s.DB.Create(&agentjobs.Run{ID: "target-run", OwnerID: target.ID, ConversationID: "other-conversation", Sequence: 1, RequestID: "request", Status: agentjobs.StatusSucceeded, QueueExpiresAt: time.Now()}).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewTransferUserResourcesLogic(context.Background(), s).TransferUserResources(&pb.TransferUserResourcesRequest{Actor: actor, UserID: source.ID, TargetUserID: target.ID, IncludeFiles: true, IncludeAI: true}); err == nil {
		t.Fatal("AI idempotency collision must fail atomically")
	}
	for _, table := range []string{"sys_file_resources", "sys_ai_conversations", "sys_ai_messages", "sys_ai_runs"} {
		var count int64
		s.DB.Table(table).Where("owner_id = ?", source.ID).Count(&count)
		if count != 1 {
			t.Fatalf("failed AI update partially moved %s", table)
		}
	}
}

func TestResourceLifecycleDeniesIncompleteFileDataScope(t *testing.T) {
	s, actor, source, target := lifecycleDB(t)
	if err := s.DB.Create(&model.SysDepartment{Name: "Allowed", Code: "allowed", Status: 1}).Error; err != nil {
		t.Fatal(err)
	}
	var department model.SysDepartment
	s.DB.First(&department)
	for _, id := range []int64{source.ID, target.ID} {
		if err := s.DB.Create(&model.SysUserDepartment{UserID: id, DepartmentID: department.ID}).Error; err != nil {
			t.Fatal(err)
		}
	}
	if err := s.DB.Model(&model.SysRoleDataScope{}).Where("authority_id = ?", 801).Update("scope", "department").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&model.SysFileResource{}).Where("owner_id = ?", source.ID).Update("department_id", 99999).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewGetUserResourcePreviewLogic(context.Background(), s).GetUserResourcePreview(&pb.UserResourcePreviewRequest{Actor: actor, UserID: source.ID}); err == nil {
		t.Fatal("partial file inventory was presented as complete")
	}
	if _, err := NewTransferUserResourcesLogic(context.Background(), s).TransferUserResources(&pb.TransferUserResourcesRequest{Actor: actor, UserID: source.ID, TargetUserID: target.ID, IncludeFiles: true}); err == nil {
		t.Fatal("out-of-scope file transferred")
	}
	var file model.SysFileResource
	if err := s.DB.First(&file).Error; err != nil || file.OwnerID != source.ID {
		t.Fatal("rejected scope changed file owner", err)
	}
}
