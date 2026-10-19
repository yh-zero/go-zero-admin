package appmembershipservicelogic

import (
	"context"
	"sync"
	"testing"
	"time"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/testmysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GO_ZERO_MYSQL_TEST_DSN enables isolated test databases; the configured
// database is never used. This checks real InnoDB locking, not SQLite emulation.
func TestMySQLConcurrentRenewalsAndRequestReplay(t *testing.T) {
	s, actor, _ := membershipFixture(t, testmysql.Open(t))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	start := make(chan struct{})
	results := make(chan error, 6)
	for _, key := range []string{"one", "one", "two", "two", "three", "three"} {
		go func(key string) {
			<-start
			in := grantInput(actor, key)
			in.Days = 1
			_, err := NewGrantAppMembershipLogic(ctx, s).GrantAppMembership(in)
			results <- err
		}(key)
	}
	close(start)
	for i := 0; i < 6; i++ {
		if err := <-results; err != nil {
			t.Fatal(err)
		}
	}
	var rows []model.AppMembershipGrant
	if err := s.DB.Order("starts_at").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("replays created %d grants", len(rows))
	}
	for i, row := range rows {
		if row.EndsAt.Sub(row.StartsAt) != 24*time.Hour {
			t.Fatal("wrong grant duration")
		}
		if i > 0 && !row.StartsAt.Equal(rows[i-1].EndsAt) {
			t.Fatal("concurrent renewal overlapped or lost time")
		}
	}
	var logs int64
	if err := s.DB.Model(&model.SysAuditLog{}).Count(&logs).Error; err != nil || logs != 3 {
		t.Fatalf("incorrect audit count=%d err=%v", logs, err)
	}
}

type mysqlMembershipContextKey struct{}

func TestMySQLRenewalSeesGrantCommittedWhileWaitingForUserLock(t *testing.T) {
	s, actor, _ := membershipFixture(t, testmysql.Open(t))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	locked, waiting, release := make(chan struct{}), make(chan struct{}), make(chan struct{})
	var releaseOnce, waitOnce sync.Once
	unlock := func() { releaseOnce.Do(func() { close(release) }) }
	defer unlock()
	if err := s.DB.Callback().Query().Before("gorm:query").Register("test:membership-user-lock", func(tx *gorm.DB) {
		if tx.Statement.Table == "app_users" && tx.Statement.Context.Value(mysqlMembershipContextKey{}) == "renewal" {
			waitOnce.Do(func() { close(waiting) })
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.DB.Callback().Query().Remove("test:membership-user-lock") })
	now := time.Now().UTC().Truncate(time.Millisecond)
	seed := model.AppMembershipGrant{UserID: 7, PlanID: 1, PlanCode: "vip", PlanVersion: 1, Entitlements: `["profile.export"]`, RequestID: "existing-concurrent-grant", Days: 1, StartsAt: now, EndsAt: now.Add(24 * time.Hour), GrantedBy: 30, Reason: "fixture"}
	firstDone := make(chan error, 1)
	go func() {
		firstDone <- s.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var user model.AppUser
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&user, 7).Error; err != nil {
				return err
			}
			if err := tx.Create(&seed).Error; err != nil {
				return err
			}
			close(locked)
			select {
			case <-release:
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		})
	}()
	wait := func(signal <-chan struct{}, phase string) {
		t.Helper()
		select {
		case <-signal:
		case <-ctx.Done():
			t.Fatalf("timed out waiting for %s", phase)
		}
	}
	wait(locked, "first grant user lock")
	type result struct {
		grant *pb.AppGrantInfo
		err   error
	}
	secondDone := make(chan result, 1)
	go func() {
		in := grantInput(actor, "renewal-after-lock")
		in.Days = 1
		g, err := NewGrantAppMembershipLogic(context.WithValue(ctx, mysqlMembershipContextKey{}, "renewal"), s).GrantAppMembership(in)
		secondDone <- result{g, err}
	}()
	wait(waiting, "renewal user lock after admin session snapshot")
	unlock()
	if err := <-firstDone; err != nil {
		t.Fatal(err)
	}
	got := <-secondDone
	if got.err != nil {
		t.Fatal(got.err)
	}
	if got.grant.StartsAt != seed.EndsAt.UTC().Format(time.RFC3339Nano) {
		t.Fatalf("renewal used stale snapshot: start=%s want=%s", got.grant.StartsAt, seed.EndsAt)
	}
}
