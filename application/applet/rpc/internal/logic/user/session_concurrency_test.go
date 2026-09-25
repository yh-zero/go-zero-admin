package userlogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/pb"
	"sync"
	"testing"
)

// SQLite serializes writes here; each goroutine still submits the same stale
// version, exercising the conditional update used by MySQL row-level locking.
func TestConcurrentLogoutOnlyRevokesOneVersion(t *testing.T) {
	svcCtx := testUserDB(t)
	user := sessionUser(t, svcCtx, "concurrent_logout", 801)
	db, _ := svcCtx.DB.DB.DB()
	db.SetMaxOpenConns(1)
	request := &pb.SessionRequest{UserID: user.ID, AuthorityId: 801, SessionVersion: user.SessionVersion}
	var group sync.WaitGroup
	failures := make(chan error, 8)
	for range 8 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := NewLogoutLogic(context.Background(), svcCtx).Logout(request)
			failures <- err
		}()
	}
	group.Wait()
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	var current model.SysUser
	if err := svcCtx.DB.First(&current, user.ID).Error; err != nil {
		t.Fatal(err)
	}
	if current.SessionVersion != user.SessionVersion+1 {
		t.Fatalf("replay revoked additional sessions: version=%d", current.SessionVersion)
	}
}
func TestConcurrentPasswordChangesOnlyOneSucceeds(t *testing.T) {
	svcCtx := testUserDB(t)
	user := sessionUser(t, svcCtx, "concurrent_password", 801)
	db, _ := svcCtx.DB.DB.DB()
	db.SetMaxOpenConns(1)
	request := &pb.ChangePasswordRequest{Session: &pb.SessionRequest{UserID: user.ID, AuthorityId: 801, SessionVersion: user.SessionVersion}, OldPassword: "before123", NewPassword: "after1234"}
	var group sync.WaitGroup
	results := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := NewChangePasswordLogic(context.Background(), svcCtx).ChangePassword(request)
			results <- err
		}()
	}
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("want exactly one successful password change, got %d", successes)
	}
}
