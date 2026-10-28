package audit

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"google.golang.org/grpc/metadata"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestSanitizeJSONRemovesSecretsAndUnstructuredValues(t *testing.T) {
	input := []byte(`{"id":3,"authorityIds":[1,2],"password":"secret","oldPassword":"old","newPassword":"new","captcha":"1234","captchaId":"captcha-secret","accessToken":"jwt-secret","phone":"private","email":"private","description":"private","children":{"id":1},"status":{"password":"secret"}}`)
	value := SanitizeJSON(input)
	if value != `{"authorityids":[1,2],"id":3}` {
		t.Fatalf("unexpected allowlist output: %s", value)
	}
	if SanitizeJSON([]byte(`not-json`)) != "{}" || SanitizeJSON([]byte(strings.Repeat("x", 16385))) != "{}" {
		t.Fatal("invalid/oversized input accepted")
	}
	if output := SanitizeJSON([]byte(`{"path":"/route?accessToken=secret","id":"jwt-secret","status":"password-secret","ids":["otp-secret"],"menuIds":"1,2","method":"PUT"}`)); output != `{"menuids":"1,2","method":"PUT"}` {
		t.Fatalf("arbitrary allowlisted string accepted: %s", output)
	}
}

func TestRecordUsesServerIdentityTimeAndRollsBackInTransaction(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&Event{}); err != nil {
		t.Fatal(err)
	}
	ctx := WithRequest(context.Background(), Request{Path: "/v1/sys/resetUserPassword", Method: "PUT", IP: "127.0.0.1", TraceID: "trace"})
	SetActor(ctx, Actor{ID: 7, Name: "admin", AuthorityID: 1})
	input := Event{ID: 999, Module: "user", Action: "resetPassword", Object: "8", CreatedAt: time.Unix(1, 0), Params: `{"id":8,"password":"secret"}`}
	before := time.Now().Add(-time.Second)
	if err := Record(ctx, db, input); err != nil {
		t.Fatal(err)
	}
	var saved Event
	if err := db.First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	if saved.ID == 999 || saved.ActorID != 7 || saved.AuthorityID != 1 || saved.ActorName != "admin" || saved.Path != "/v1/sys/resetUserPassword" || saved.CreatedAt.Before(before) || saved.Params != `{"id":8}` {
		t.Fatalf("unexpected persisted event: %+v", saved)
	}
	rollback := errors.New("business rollback")
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := Record(ctx, tx, input); err != nil {
			return err
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatalf("missing rollback: %v", err)
	}
	var count int64
	if err := db.Model(&Event{}).Count(&count).Error; err != nil || count != 1 {
		t.Fatalf("audit transaction did not roll back count=%d err=%v", count, err)
	}
}

func TestMetadataOnlyComesFromVerifiedContext(t *testing.T) {
	ctx := WithRequest(context.Background(), Request{Path: "/v1/sys/register", Method: "POST", UserAgent: "测试设备"})
	SetActor(ctx, Actor{ID: 5, Name: "管理员", AuthorityID: 1})
	ctx = metadata.NewOutgoingContext(ctx, metadata.Pairs("x-audit-actor-id", "999"))
	outgoing := OutgoingContext(ctx)
	md, _ := metadata.FromOutgoingContext(outgoing)
	incoming := metadata.NewIncomingContext(context.Background(), md)
	if actor := ActorFromContext(incoming); actor.ID != 5 || actor.Name != "管理员" || actor.AuthorityID != 1 {
		t.Fatalf("spoofed actor: %+v", actor)
	}
	if RequestFromContext(incoming).Path != "/v1/sys/register" {
		t.Fatal("request metadata lost")
	}
	if RequestFromContext(incoming).UserAgent != "测试设备" {
		t.Fatal("user-agent metadata lost")
	}
	var params map[string]any
	if err := json.Unmarshal([]byte(SanitizeJSON([]byte(`{"ID":4,"pageNo":2}`))), &params); err != nil || params["id"] != float64(4) {
		t.Fatal("case insensitive fields failed")
	}
}
