package agenttoolslogic

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/casbin/casbin/v2"
	cm "github.com/casbin/casbin/v2/model"
	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	base "go-zero-admin/pkg/model"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func fixture(t *testing.T) (*svc.ServiceContext, *pb.SessionRequest) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{DisableForeignKeyConstraintWhenMigrating: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sql, _ := db.DB()
	sql.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = sql.Close() })
	if err := db.AutoMigrate(&model.SysUser{}, &model.SysAuthority{}, &model.SysUserAuthority{}, &model.SysDeviceSession{}, &model.SysRoleDataScope{}, &model.SysRoleScopeDepartment{}, &model.SysFileResource{}, &model.SysFileReference{}, &model.SysAuditLog{}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{1, 2} {
		if err := db.Create(&model.SysAuthority{AuthorityId: id, AuthorityName: "role"}).Error; err != nil {
			t.Fatal(err)
		}
	}
	for _, u := range []model.SysUser{{MODEL_BASE: base.MODEL_BASE{ID: 10}, Username: "member", AuthorityId: 2, Enable: 1, SessionVersion: 1}, {MODEL_BASE: base.MODEL_BASE{ID: 30}, Username: "admin", AuthorityId: 1, Enable: 1, SessionVersion: 1}} {
		if err := db.Omit("Authority", "Authorities").Create(&u).Error; err != nil {
			t.Fatal(err)
		}
		if err := db.Create(&model.SysUserAuthority{SysUserId: u.ID, SysAuthorityAuthorityId: u.AuthorityId}).Error; err != nil {
			t.Fatal(err)
		}
	}
	m, err := cm.NewModelFromString("[request_definition]\nr = sub, obj, act\n[policy_definition]\np = sub, obj, act\n[policy_effect]\ne = some(where (p.eft == allow))\n[matchers]\nm = r.sub == p.sub && r.obj == p.obj && r.act == p.act")
	if err != nil {
		t.Fatal(err)
	}
	enforcer, err := casbin.NewSyncedCachedEnforcer(m)
	if err != nil {
		t.Fatal(err)
	}
	enforcer.EnableCache(false)
	for _, p := range [][]string{{"2", "/v1/ai/runs", "POST"}, {"2", "/v1/ai/info", "GET"}, {"2", "/v1/sys/files/list", "GET"}, {"2", "/v1/sys/audit/getAuditLogList", "GET"}, {"1", "/v1/ai/runs", "POST"}, {"1", "/v1/sys/files/list", "GET"}} {
		if _, err := enforcer.AddPolicy(p); err != nil {
			t.Fatal(err)
		}
	}
	return &svc.ServiceContext{DB: &orm.DB{DB: db}, Casbin: enforcer}, &pb.SessionRequest{UserID: 10, AuthorityId: 2, SessionVersion: 1}
}

func TestToolsRecheckAuthorizationAndScope(t *testing.T) {
	s, a := fixture(t)
	ctx := context.Background()
	for _, f := range []model.SysFileResource{{ID: 1, OwnerID: 10, ObjectKey: "private-secret-a", Name: "mine", Status: "active"}, {ID: 2, OwnerID: 30, ObjectKey: "private-secret-b", Name: "other", Status: "active"}} {
		if err := s.DB.Create(&f).Error; err != nil {
			t.Fatal(err)
		}
	}
	file := NewGetAgentFileStatusLogic(ctx, s)
	result, err := file.GetAgentFileStatus(&pb.AgentToolRequest{Actor: a, ArgumentsJson: `{"fileId":1}`})
	if err != nil || strings.Contains(result.Content, "private-secret") {
		t.Fatal(result, err)
	}
	if _, err := file.GetAgentFileStatus(&pb.AgentToolRequest{Actor: a, ArgumentsJson: `{"fileId":2}`}); err == nil {
		t.Fatal("cross-owner file readable")
	}
	if _, err := s.Casbin.RemovePolicy("2", "/v1/sys/files/list", "GET"); err != nil {
		t.Fatal(err)
	}
	if _, err := file.GetAgentFileStatus(&pb.AgentToolRequest{Actor: a, ArgumentsJson: `{"fileId":1}`}); err == nil {
		t.Fatal("revoked tool grant still usable")
	}
	if err := s.DB.Model(&model.SysUser{}).Where("id=10").Update("enable", 2).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := NewQueryAgentAuditLogic(ctx, s).QueryAgentAudit(&pb.AgentToolRequest{Actor: a, ArgumentsJson: `{}`}); err == nil {
		t.Fatal("disabled user used captured tool")
	}
}
func TestToolsRejectUnexpectedDuplicateAndNullArguments(t *testing.T) {
	for _, raw := range []string{`{"fileId":1,"fileId":2}`, `{"fileId":1,"FileId":2}`, `{"FileID":1}`, `{"fileId":null}`, `{"userId":30}`, `{} {}`, `[]`, `{"fileId":"1"}`, `{"fileId":1.2}`, string([]byte{'{', '"', 0xff, '"', ':', '1', '}'})} {
		var args struct {
			FileID int64 `json:"fileId"`
		}
		if decodeArgs(json.RawMessage(raw), &args) == nil {
			t.Fatal("accepted", raw)
		}
	}
	if decodeArgs(json.RawMessage(`{"fileId":1}`), &struct {
		FileID int64 `json:"fileId"`
	}{}) != nil {
		t.Fatal("valid arg rejected")
	}
}
func TestAuditFullAggregateAndPrivateFields(t *testing.T) {
	s, _ := fixture(t)
	now := time.Now().UTC()
	for i := 0; i < 25; i++ {
		if err := s.DB.Create(&model.SysAuditLog{Module: "files", Action: "read", Result: "failure", EventType: "operation", Params: "secret-body", IP: "secret-ip", CreatedAt: now}).Error; err != nil {
			t.Fatal(err)
		}
	}
	r, err := queryAudit(context.Background(), s, json.RawMessage(`{"result":"failure"}`))
	if err != nil {
		t.Fatal(err)
	}
	var data struct {
		Total    int64
		ByModule []struct{ Count int64 }
		Recent   []any
	}
	if err := json.Unmarshal([]byte(r.Content), &data); err != nil {
		t.Fatal(err)
	}
	if data.Total != 25 || len(data.Recent) != 20 || len(data.ByModule) != 1 || data.ByModule[0].Count != 25 {
		t.Fatal(r.Content)
	}
	if strings.Contains(r.Content, "secret-") {
		t.Fatal("private audit data exported")
	}
	if _, _, err := auditRange(auditArgs{StartTime: now.Add(-32 * 24 * time.Hour).Format(time.RFC3339)}, now); err == nil {
		t.Fatal("unbounded query allowed")
	}
}
func TestDeviceToolAlwaysUsesSelfEvenForAdmin(t *testing.T) {
	s, _ := fixture(t)
	a := &pb.SessionRequest{UserID: 30, AuthorityId: 1, SessionVersion: 1}
	for _, uid := range []int64{10, 30} {
		if err := s.DB.Create(&model.SysDeviceSession{ID: strings.Repeat("a", int(uid/10)), UserID: uid, AuthorityID: map[int64]int64{10: 2, 30: 1}[uid], SessionVersion: 1, IP: "secret-ip", UserAgent: "secret-ua Chrome/123", ExpiresAt: time.Now().Add(time.Hour)}).Error; err != nil {
			t.Fatal(err)
		}
	}
	r, err := myDevices(context.Background(), s, a, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	var data struct{ Total int64 }
	_ = json.Unmarshal([]byte(r.Content), &data)
	if data.Total != 1 || strings.Contains(r.Content, "secret") || strings.Contains(r.Content, "userId") {
		t.Fatal(r.Content)
	}
}
