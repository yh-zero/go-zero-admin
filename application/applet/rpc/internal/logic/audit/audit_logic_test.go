package auditlogic

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/orm"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func testAuditDB(t *testing.T) *svc.ServiceContext {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&model.SysAuditLog{}); err != nil {
		t.Fatal(err)
	}
	return &svc.ServiceContext{DB: &orm.DB{DB: db}}
}

func TestRecordAuditRejectsInvalidAndSanitizesCaller(t *testing.T) {
	s := testAuditDB(t)
	l := NewRecordAuditLogic(context.Background(), s)
	for _, req := range []*pb.RecordAuditRequest{nil, {}, {Log: &pb.AuditLog{Module: "user", Action: "reset", Result: "invalid"}}, {Log: &pb.AuditLog{Module: "user", Action: "reset", ActorID: -1}}} {
		if _, err := l.RecordAudit(req); err == nil {
			t.Fatal("invalid event accepted")
		}
	}
	_, err := l.RecordAudit(&pb.RecordAuditRequest{Log: &pb.AuditLog{ID: 999, ActorID: 2, Module: "user", Action: "resetPassword", Params: `{"id":4,"password":"private","accessToken":"token"}`, CreatedAt: "2000-01-01T00:00:00Z"}})
	if err != nil {
		t.Fatal(err)
	}
	var event model.SysAuditLog
	if err := s.DB.First(&event).Error; err != nil {
		t.Fatal(err)
	}
	if event.ID == 999 || event.CreatedAt.Year() == 2000 || event.Params != `{"id":4}` {
		t.Fatalf("untrusted fields persisted: %+v", event)
	}
}

func TestAuditQueryFiltersPaginationAndTime(t *testing.T) {
	s := testAuditDB(t)
	base := time.Date(2026, 10, 2, 10, 0, 0, 0, time.UTC)
	for i, event := range []audit.Event{
		{ActorID: 1, ActorName: "admin", Module: "user", Action: "login", EventType: "login", Result: "failure", Params: "{}"},
		{ActorID: 1, ActorName: "admin", Module: "user", Action: "resetPassword", EventType: "operation", Result: "success", Params: "{}"},
		{ActorID: 2, ActorName: "other", Module: "menu", Action: "delete", EventType: "operation", Result: "success", Params: "{}"},
	} {
		event.CreatedAt = base.Add(time.Duration(i) * time.Minute)
		if err := s.DB.Create(&event).Error; err != nil {
			t.Fatal(err)
		}
	}
	l := NewGetAuditLogListLogic(context.Background(), s)
	zone := time.FixedZone("Asia/Shanghai", 8*60*60)
	response, err := l.GetAuditLogList(&pb.GetAuditLogListRequest{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 1}, Module: "user", ActorID: 1, ActorName: "admin", EventType: "operation", Result: "success", StartTime: base.In(zone).Format(time.RFC3339), EndTime: base.Add(time.Minute).In(zone).Format(time.RFC3339)})
	if err != nil {
		t.Fatal(err)
	}
	if response.Total != 1 || len(response.List) != 1 || response.List[0].Action != "resetPassword" {
		t.Fatalf("unexpected filtered result: %+v", response)
	}
	response, err = l.GetAuditLogList(&pb.GetAuditLogListRequest{PageRequest: &pb.PageRequest{PageNo: 2, PageSize: 1}})
	if err != nil || response.Total != 3 || response.List[0].Action != "resetPassword" {
		t.Fatalf("pagination unstable: %+v err=%v", response, err)
	}
	for _, req := range []*pb.GetAuditLogListRequest{
		nil, {}, {PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 1}, StartTime: "2026-10-02"},
		{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 1}, StartTime: base.Add(time.Hour).Format(time.RFC3339), EndTime: base.Format(time.RFC3339)},
		{PageRequest: &pb.PageRequest{PageNo: 1, PageSize: 1}, ActorID: -1},
	} {
		if _, err := l.GetAuditLogList(req); err == nil {
			t.Fatal("invalid query accepted")
		}
	}
}
