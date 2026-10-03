package agenttoolslogic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	base "go-zero-admin/pkg/model"
	"gorm.io/gorm"
)

type toolCall func(*pb.AgentToolRequest) (*pb.AgentToolResult, error)

func facades(s *svc.ServiceContext) []struct {
	name, arguments, table string
	call                   toolCall
} {
	ctx := context.Background()
	return []struct {
		name, arguments, table string
		call                   toolCall
	}{
		{"audit", "{}", "sys_audit_logs", NewQueryAgentAuditLogic(ctx, s).QueryAgentAudit},
		{"file", `{"fileId":1}`, "sys_file_resources", NewGetAgentFileStatusLogic(ctx, s).GetAgentFileStatus},
		{"devices", "{}", "sys_device_sessions", NewListAgentDevicesLogic(ctx, s).ListAgentDevices},
	}
}

func seedFacadeData(t *testing.T, s *svc.ServiceContext, actor *pb.SessionRequest) {
	t.Helper()
	now := time.Now().UTC()
	rows := []any{
		&model.SysFileResource{ID: 1, OwnerID: actor.UserID, ObjectKey: "private-storage-secret", Name: "报告.txt", Mime: "text/plain", Size: 12, Status: "active", Visibility: "private"},
		&model.SysFileReference{FileID: 1, ObjectType: "report", ObjectID: "private-reference-secret"},
		&model.SysAuditLog{ActorName: "member", Module: "files", Action: "delete", Result: "failure", StatusCode: 200, EventType: "operation", Params: "private-body-secret", IP: "private-ip-secret", Object: "private-object-secret", TraceID: "private-trace-secret", CreatedAt: now.Add(-time.Minute)},
		&model.SysDeviceSession{ID: "member-device-secret", UserID: actor.UserID, AuthorityID: actor.AuthorityId, SessionVersion: actor.SessionVersion, UserAgent: "private-agent-secret Chrome/123", IP: "private-ip-secret", ExpiresAt: now.Add(time.Hour)},
		&model.SysDeviceSession{ID: "other-device-secret", UserID: 30, AuthorityID: 1, SessionVersion: 1, UserAgent: "Firefox/123", ExpiresAt: now.Add(time.Hour)},
	}
	for _, row := range rows {
		if err := s.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
}

func TestThreeFacadesReturnOnlySafeBusinessFields(t *testing.T) {
	s, actor := fixture(t)
	seedFacadeData(t, s, actor)
	actor.SessionID = "member-device-secret"
	for _, endpoint := range facades(s) {
		t.Run(endpoint.name, func(t *testing.T) {
			result, err := endpoint.call(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: endpoint.arguments})
			if err != nil || result == nil || result.Count != 1 || result.Summary == "" || !json.Valid([]byte(result.Content)) {
				t.Fatalf("invalid RPC tool DTO: result=%v err=%v", result, err)
			}
			for _, field := range []string{"secret", "\"params\"", "\"ip\"", "\"objectKey\"", "\"sessionId\"", "\"userId\"", "\"userAgent\"", "\"ownerId\"", "\"departmentId\"", "\"traceId\""} {
				if strings.Contains(result.Content, field) {
					t.Fatalf("private tool field exported: %s", result.Content)
				}
			}
			var data map[string]any
			if err := json.Unmarshal([]byte(result.Content), &data); err != nil {
				t.Fatal(err)
			}
			switch endpoint.name {
			case "audit":
				if data["total"] != float64(1) || len(data["recent"].([]any)) != 1 {
					t.Fatal(result.Content)
				}
			case "file":
				if data["references"] != float64(1) || data["visibility"] != "private" || !strings.Contains(data["explanation"].(string), "移除引用") {
					t.Fatal(result.Content)
				}
			case "devices":
				items := data["items"].([]any)
				if data["total"] != float64(1) || items[0].(map[string]any)["client"] != "Chrome" || items[0].(map[string]any)["current"] != true {
					t.Fatal("devices were not restricted to actor", result.Content)
				}
			}
		})
	}
}

func TestFacadesRejectMissingAndForgedActor(t *testing.T) {
	s, _ := fixture(t)
	for _, endpoint := range facades(s) {
		t.Run(endpoint.name, func(t *testing.T) {
			if result, err := endpoint.call(nil); err == nil || result != nil {
				t.Fatal("nil request accepted")
			}
			for _, actor := range []*pb.SessionRequest{
				nil, {}, {UserID: 10, AuthorityId: 2}, {UserID: 10, AuthorityId: 1, SessionVersion: 1},
				{UserID: 30, AuthorityId: 2, SessionVersion: 1}, {UserID: 10, AuthorityId: 2, SessionVersion: 2},
				{UserID: 10, AuthorityId: 2, SessionVersion: 1, SessionID: "missing-device"},
			} {
				result, err := endpoint.call(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: endpoint.arguments})
				if err == nil || result != nil {
					t.Fatalf("forged actor accepted: %v result=%v", actor, result)
				}
			}
		})
	}
}

func TestFacadesNeverAllowModelToSelectActor(t *testing.T) {
	s, actor := fixture(t)
	for _, endpoint := range facades(s) {
		t.Run(endpoint.name, func(t *testing.T) {
			for _, raw := range []string{`{"userId":30}`, `{"actor":{"userID":30,"authorityId":1}}`, `{"sessionID":"other-device"}`} {
				result, err := endpoint.call(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: raw})
				if err == nil || result != nil {
					t.Fatal("model-controlled actor accepted", raw, result)
				}
			}
		})
	}
}

func TestFacadesRejectExecutionGrantRevokedDuringBusinessRead(t *testing.T) {
	for _, name := range []string{"audit", "file", "devices"} {
		t.Run(name, func(t *testing.T) {
			s, actor := fixture(t)
			seedFacadeData(t, s, actor)
			var selected struct {
				name, arguments, table string
				call                   toolCall
			}
			for _, endpoint := range facades(s) {
				if endpoint.name == name {
					selected = endpoint
				}
			}
			revoked := false
			if err := s.DB.Callback().Query().After("gorm:query").Register("test_revoke_agent", func(query *gorm.DB) {
				if !revoked && query.Statement.Table == selected.table {
					revoked = true
					if _, err := s.Casbin.RemovePolicy("2", "/v1/ai/runs", "POST"); err != nil {
						t.Error(err)
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			result, err := selected.call(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: selected.arguments})
			if !revoked || err == nil || result != nil {
				t.Fatalf("late revocation leaked output: revoked=%v result=%v err=%v", revoked, result, err)
			}
		})
	}
}

func TestFileFacadeRechecksSessionAndBusinessGrantAfterSQL(t *testing.T) {
	for _, revoke := range []string{"session", "file-grant"} {
		t.Run(revoke, func(t *testing.T) {
			s, actor := fixture(t)
			seedFacadeData(t, s, actor)
			changed := false
			if err := s.DB.Callback().Query().After("gorm:query").Register("test_revoke_file", func(query *gorm.DB) {
				if !changed && query.Statement.Table == "sys_file_resources" && query.RowsAffected > 0 {
					changed = true
					if revoke == "session" {
						if err := s.DB.Model(&model.SysUser{}).Where("id=?", actor.UserID).Update("session_version", 2).Error; err != nil {
							t.Error(err)
						}
					} else if _, err := s.Casbin.RemovePolicy("2", "/v1/sys/files/list", "GET"); err != nil {
						t.Error(err)
					}
				}
			}); err != nil {
				t.Fatal(err)
			}
			result, err := NewGetAgentFileStatusLogic(context.Background(), s).GetAgentFileStatus(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: `{"fileId":1}`})
			if !changed || err == nil || result != nil {
				t.Fatalf("post-query authorization not checked: result=%v err=%v", result, err)
			}
		})
	}
}

func TestDeviceFacadeRechecksSessionAfterBusinessRead(t *testing.T) {
	s, actor := fixture(t)
	seedFacadeData(t, s, actor)
	actor.SessionID = "member-device-secret"
	revoked := false
	if err := s.DB.Callback().Query().After("gorm:query").Register("test_revoke_device_session", func(query *gorm.DB) {
		if !revoked && query.Statement.Table == "sys_device_sessions" && query.RowsAffected > 0 {
			revoked = true
			if err := s.DB.Model(&model.SysDeviceSession{}).Where("id = ?", actor.SessionID).Update("revoked_at", time.Now().UTC()).Error; err != nil {
				t.Error(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	result, err := NewListAgentDevicesLogic(context.Background(), s).ListAgentDevices(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: "{}"})
	if !revoked || err == nil || result != nil {
		t.Fatalf("revoked session leaked devices: revoked=%v result=%v err=%v", revoked, result, err)
	}
}

func TestFileFacadeUsesDepartmentScopeInSQL(t *testing.T) {
	s, actor := fixture(t)
	if err := s.DB.AutoMigrate(&model.SysDepartment{}, &model.SysUserDepartment{}); err != nil {
		t.Fatal(err)
	}
	for _, row := range []any{
		&model.SysDepartment{MODEL_BASE: base.MODEL_BASE{ID: 7}, Name: "所属部门", Code: "dept-7", Status: 1},
		&model.SysDepartment{MODEL_BASE: base.MODEL_BASE{ID: 8}, Name: "其他部门", Code: "dept-8", Status: 1},
		&model.SysUserDepartment{UserID: actor.UserID, DepartmentID: 7},
		&model.SysRoleDataScope{AuthorityID: actor.AuthorityId, Scope: "department"},
		&model.SysFileResource{ID: 1, OwnerID: 30, DepartmentID: 7, ObjectKey: "department-file", Name: "部门报告", Status: "active"},
		&model.SysFileResource{ID: 2, OwnerID: actor.UserID, DepartmentID: 8, ObjectKey: "outside-department-file", Name: "其他报告", Status: "active"},
	} {
		if err := s.DB.Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	logic := NewGetAgentFileStatusLogic(context.Background(), s)
	if result, err := logic.GetAgentFileStatus(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: `{"fileId":1}`}); err != nil || result == nil {
		t.Fatal("authorized department file rejected", result, err)
	}
	if result, err := logic.GetAgentFileStatus(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: `{"fileId":2}`}); err == nil || result != nil {
		t.Fatal("department scope was bypassed", result, err)
	}
}

func TestFileFacadeRejectsDataScopeNarrowedDuringRead(t *testing.T) {
	s, actor := fixture(t)
	if err := s.DB.Create(&model.SysRoleDataScope{AuthorityID: actor.AuthorityId, Scope: "all"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Create(&model.SysFileResource{ID: 1, OwnerID: 30, ObjectKey: "other-owner-file", Name: "报告", Status: "active"}).Error; err != nil {
		t.Fatal(err)
	}
	changed := false
	if err := s.DB.Callback().Query().After("gorm:query").Register("test_narrow_scope", func(query *gorm.DB) {
		if !changed && query.Statement.Table == "sys_file_resources" && query.RowsAffected > 0 {
			changed = true
			if err := s.DB.Model(&model.SysRoleDataScope{}).Where("authority_id=?", actor.AuthorityId).Update("scope", "self").Error; err != nil {
				t.Error(err)
			}
		}
	}); err != nil {
		t.Fatal(err)
	}
	result, err := NewGetAgentFileStatusLogic(context.Background(), s).GetAgentFileStatus(&pb.AgentToolRequest{Actor: actor, ArgumentsJson: `{"fileId":1}`})
	if !changed || err == nil || result != nil {
		t.Fatalf("narrowed SQL range leaked file: result=%v err=%v", result, err)
	}
}
