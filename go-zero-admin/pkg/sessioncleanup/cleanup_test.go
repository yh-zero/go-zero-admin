package sessioncleanup

import (
	"context"
	"github.com/glebarez/sqlite"
	"go-zero-admin/pkg/audit"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestCleanupDryRunThenBatchesKeepLiveAndRecentRevocation(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, _ := db.DB()
	t.Cleanup(func() { _ = sqlDB.Close() })
	if err := db.AutoMigrate(&Session{}, &audit.Event{}); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC)
	old := now.AddDate(0, 0, -40)
	recent := now.AddDate(0, 0, -2)
	rows := []Session{
		{ID: "old-expired", CreatedAt: old, ExpiresAt: old}, {ID: "old-revoked", CreatedAt: old, ExpiresAt: now.Add(time.Hour), RevokedAt: &old},
		{ID: "live", CreatedAt: old, ExpiresAt: now.Add(time.Hour)}, {ID: "recent-expired", CreatedAt: old, ExpiresAt: recent},
		{ID: "recent-revoked-old-expiry", CreatedAt: old, ExpiresAt: old, RevokedAt: &recent}, {ID: "boundary", CreatedAt: old, ExpiresAt: now.AddDate(0, 0, -30)},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if err := audit.Record(context.Background(), db, audit.Event{Module: "session", Action: "revoke", Object: "old-revoked"}); err != nil {
		t.Fatal(err)
	}
	options := Options{RetentionDays: 30, BatchSize: 1, Now: now}
	preview, err := Run(context.Background(), db, options)
	if err != nil || !preview.DryRun || preview.Eligible != 2 || preview.Deleted != 0 {
		t.Fatalf("dry-run must identify two old terminal sessions without deleting: %+v %v", preview, err)
	}
	var count int64
	db.Model(&Session{}).Count(&count)
	if count != 6 {
		t.Fatal("dry run changed sessions")
	}
	options.Apply = true
	actual, err := Run(context.Background(), db, options)
	if err != nil || actual.DryRun || actual.Deleted != 2 || actual.Batches != 2 {
		t.Fatalf("batch cleanup failed: %+v %v", actual, err)
	}
	var survivors []Session
	db.Order("id").Find(&survivors)
	if len(survivors) != 4 {
		t.Fatalf("valid/recent session removed: %+v", survivors)
	}
	db.Model(&audit.Event{}).Count(&count)
	if count != 1 {
		t.Fatal("cleanup modified audit trail")
	}
}
func TestCleanupRequiresExplicitRetentionAndBatch(t *testing.T) {
	for _, options := range []Options{{}, {RetentionDays: 30}, {BatchSize: 100}, {RetentionDays: -1, BatchSize: 100}, {RetentionDays: 30, BatchSize: 5001}} {
		if _, err := Run(context.Background(), nil, options); err == nil {
			t.Fatalf("invalid options accepted: %+v", options)
		}
	}
}
