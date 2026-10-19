package appaccountservicelogic

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/appauth"
	"go-zero-admin/pkg/result/xerr"
	"go-zero-admin/pkg/testmysql"

	"gorm.io/gorm"
)

// These opt-in tests use disposable databases and MySQL 8's lock-wait view to
// prove the competing requests actually overlap while holding InnoDB row locks.
func TestMySQLConcurrentRefreshReplayRevokesSession(t *testing.T) {
	s, _ := accountFixtureWithDB(t, false, testmysql.Open(t))
	session, original := seedMySQLRefreshSession(t, s)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	locked, release := pauseMySQLAccountUserLock(t, s.DB.DB, "first-refresh")
	defer release()
	firstDone := make(chan mysqlAccountRefreshResult, 1)
	go func() {
		tokens, err := NewRefreshAppTokenLogic(context.WithValue(ctx, mysqlAccountOperationKey{}, "first-refresh"), s).
			RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: original.RefreshToken})
		firstDone <- mysqlAccountRefreshResult{tokens: tokens, err: err}
	}()
	waitMySQLAccountSignal(t, ctx, locked)
	secondDone := make(chan mysqlAccountRefreshResult, 1)
	go func() {
		tokens, err := NewRefreshAppTokenLogic(ctx, s).RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: original.RefreshToken})
		secondDone <- mysqlAccountRefreshResult{tokens: tokens, err: err}
	}()
	waitMySQLAccountLockContention(t, ctx, s.DB.DB)
	release()
	first, second := <-firstDone, <-secondDone
	if first.err != nil || first.tokens == nil {
		t.Fatalf("first refresh failed: %v", first.err)
	}
	assertMySQLAccountUnauthorized(t, second.err)
	if second.tokens != nil {
		t.Fatal("concurrent replay returned tokens")
	}
	var persisted model.AppSession
	if err := s.DB.First(&persisted, "id = ?", session.ID).Error; err != nil {
		t.Fatal(err)
	}
	if persisted.RefreshGeneration != 2 || persisted.RevokedAt == nil || persisted.RefreshHash != appauth.Digest(first.tokens.RefreshToken) || persisted.PreviousRefreshHash != appauth.Digest(original.RefreshToken) {
		t.Fatalf("concurrent refresh did not rotate exactly once and revoke the session: generation=%d revoked=%v", persisted.RefreshGeneration, persisted.RevokedAt != nil)
	}
	assertMySQLAccountTokensRejected(t, ctx, s, original)
	assertMySQLAccountTokensRejected(t, ctx, s, first.tokens)
}

func TestMySQLDisableAndRefreshSerializeOnUserLock(t *testing.T) {
	for _, firstOperation := range []string{"disable", "refresh"} {
		t.Run(firstOperation+" first", func(t *testing.T) {
			s, actor := accountFixtureWithDB(t, false, testmysql.Open(t))
			session, original := seedMySQLRefreshSession(t, s)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			locked, release := pauseMySQLAccountUserLock(t, s.DB.DB, firstOperation)
			defer release()
			refreshDone := make(chan mysqlAccountRefreshResult, 1)
			disableDone := make(chan error, 1)
			startRefresh := func() {
				go func() {
					tokens, err := NewRefreshAppTokenLogic(context.WithValue(ctx, mysqlAccountOperationKey{}, "refresh"), s).
						RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: original.RefreshToken})
					refreshDone <- mysqlAccountRefreshResult{tokens: tokens, err: err}
				}()
			}
			startDisable := func() {
				go func() {
					_, err := NewSetAppUserStatusLogic(context.WithValue(ctx, mysqlAccountOperationKey{}, "disable"), s).
						SetAppUserStatus(&pb.AppStatusRequest{Actor: actor, UserID: session.UserID, Status: "disabled", Reason: "concurrent refresh regression"})
					disableDone <- err
				}()
			}
			if firstOperation == "disable" {
				startDisable()
				waitMySQLAccountSignal(t, ctx, locked)
				startRefresh()
			} else {
				startRefresh()
				waitMySQLAccountSignal(t, ctx, locked)
				startDisable()
			}
			waitMySQLAccountLockContention(t, ctx, s.DB.DB)
			release()
			refreshResult, disableErr := <-refreshDone, <-disableDone
			if disableErr != nil {
				t.Fatalf("disabling the account failed: %v", disableErr)
			}
			if firstOperation == "disable" {
				assertMySQLAccountUnauthorized(t, refreshResult.err)
				if refreshResult.tokens != nil {
					t.Fatal("refresh issued tokens after disable committed")
				}
			} else {
				if refreshResult.err != nil || refreshResult.tokens == nil {
					t.Fatalf("refresh holding the first lock failed: %v", refreshResult.err)
				}
				assertMySQLAccountTokensRejected(t, ctx, s, refreshResult.tokens)
			}
			assertMySQLAccountTokensRejected(t, ctx, s, original)
			var user model.AppUser
			if err := s.DB.First(&user, session.UserID).Error; err != nil {
				t.Fatal(err)
			}
			if user.Status != "disabled" || user.SessionVersion != 2 {
				t.Fatalf("disable lost its status or session version: status=%s version=%d", user.Status, user.SessionVersion)
			}
			var persisted model.AppSession
			if err := s.DB.First(&persisted, "id = ?", session.ID).Error; err != nil {
				t.Fatal(err)
			}
			if persisted.RevokedAt == nil {
				t.Fatal("disable left the concurrently refreshed session active")
			}
			var auditCount int64
			if err := s.DB.Model(&model.SysAuditLog{}).Where("action = ?", "setAppUserStatus:disabled").Count(&auditCount).Error; err != nil || auditCount != 1 {
				t.Fatalf("disable audit count=%d err=%v", auditCount, err)
			}
		})
	}
}

type mysqlAccountOperationKey struct{}
type mysqlAccountRefreshResult struct {
	tokens *pb.AppAuthTokens
	err    error
}

func seedMySQLRefreshSession(t *testing.T, s *svc.ServiceContext) (model.AppSession, *pb.AppAuthTokens) {
	t.Helper()
	var innodbTables int64
	if err := s.DB.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = DATABASE() AND table_name IN ('app_users', 'app_sessions') AND engine = 'InnoDB'").Scan(&innodbTables).Error; err != nil || innodbTables != 2 {
		t.Fatalf("account concurrency tests require InnoDB user/session tables: count=%d err=%v", innodbTables, err)
	}
	user, _ := seedActive(t, s, 1)
	var session model.AppSession
	if err := s.DB.Where("user_id = ?", user.ID).First(&session).Error; err != nil {
		t.Fatal(err)
	}
	tokens, err := issueTokens(context.Background(), s, user, &session)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.DB.Model(&session).Update("refresh_hash", session.RefreshHash).Error; err != nil {
		t.Fatal(err)
	}
	return session, tokens
}

func pauseMySQLAccountUserLock(t *testing.T, db *gorm.DB, operation string) (<-chan struct{}, func()) {
	t.Helper()
	locked, release := make(chan struct{}), make(chan struct{})
	var pauseOnce, releaseOnce sync.Once
	const callback = "test:pause-account-user-lock"
	if err := db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		if tx.Error == nil && tx.Statement.Table == "app_users" && tx.Statement.Context.Value(mysqlAccountOperationKey{}) == operation {
			pauseOnce.Do(func() {
				close(locked)
				select {
				case <-release:
				case <-tx.Statement.Context.Done():
					tx.AddError(tx.Statement.Context.Err())
				}
			})
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Callback().Query().Remove(callback) })
	return locked, func() { releaseOnce.Do(func() { close(release) }) }
}

func waitMySQLAccountSignal(t *testing.T, ctx context.Context, ready <-chan struct{}) {
	t.Helper()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("timed out waiting for the first request to lock the App user")
	}
}

func waitMySQLAccountLockContention(t *testing.T, ctx context.Context, db *gorm.DB) {
	t.Helper()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var count int64
		err := db.WithContext(ctx).Raw(`SELECT COUNT(*)
			FROM performance_schema.data_lock_waits AS w
			JOIN performance_schema.data_locks AS l
			  ON l.ENGINE = w.ENGINE AND l.ENGINE_LOCK_ID = w.REQUESTING_ENGINE_LOCK_ID
			WHERE l.OBJECT_SCHEMA = DATABASE() AND l.OBJECT_NAME = 'app_users'`).Scan(&count).Error
		if err != nil {
			t.Fatalf("cannot observe MySQL row-lock contention: %v", err)
		}
		if count > 0 {
			return
		}
		select {
		case <-ticker.C:
		case <-ctx.Done():
			t.Fatal("competing request never waited on the InnoDB App user row lock")
		}
	}
}

func assertMySQLAccountUnauthorized(t *testing.T, err error) {
	t.Helper()
	var codeErr *xerr.CodeError
	if !errors.As(err, &codeErr) || codeErr.GetErrCode() != xerr.TOKEN_EXPIRE_ERROR {
		t.Fatalf("expected expired/revoked token rejection, got %v", err)
	}
}

func assertMySQLAccountTokensRejected(t *testing.T, ctx context.Context, s *svc.ServiceContext, tokens *pb.AppAuthTokens) {
	t.Helper()
	_, err := NewCheckAppSessionLogic(ctx, s).CheckAppSession(&pb.AppAccessRequest{AccessToken: tokens.AccessToken})
	assertMySQLAccountUnauthorized(t, err)
	_, err = NewRefreshAppTokenLogic(ctx, s).RefreshAppToken(&pb.AppRefreshRequest{RefreshToken: tokens.RefreshToken})
	assertMySQLAccountUnauthorized(t, err)
}
