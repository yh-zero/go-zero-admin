package model

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type permissionTestCache struct{ cache.Cache }

func (permissionTestCache) DelCtx(context.Context, ...string) error { return nil }

func permissionTestDB(t *testing.T) (*sql.DB, sqlx.SqlConn) {
	t.Helper()
	dsn := os.Getenv("GOZERO_ADMIN_TEST_DSN")
	if dsn == "" {
		t.Skip("需要隔离的 GOZERO_ADMIN_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db, sqlx.NewSqlConnFromDB(db)
}

func TestRoleMenuPolicyGrantAndRevoke(t *testing.T) {
	db, conn := permissionTestDB(t)
	code := fmt.Sprintf("sync_role_%d", time.Now().UnixNano())
	username := fmt.Sprintf("sync_user_%d", time.Now().UnixNano())
	var roleID, userID int64
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM casbin_rule WHERE v0=? OR v1=?", code, code)
		_, _ = db.Exec("DELETE FROM sys_role_menu WHERE role_id=?", roleID)
		_, _ = db.Exec("DELETE FROM sys_role WHERE id=? AND code=?", roleID, code)
		_, _ = db.Exec("DELETE FROM sys_user WHERE id=? AND username=?", userID, username)
	})
	roleModel := &customSysRoleModel{defaultSysRoleModel: &defaultSysRoleModel{CachedConn: sqlc.NewConnWithCache(conn, permissionTestCache{}), table: "`sys_role`"}}
	role := &SysRole{Name: code, Code: code, DataScope: "SELF", Status: 1}
	if err := roleModel.CreateWithMenus(t.Context(), role, []int64{9001320}); err != nil {
		t.Fatal(err)
	}
	roleID = int64(role.Id)
	result, err := db.ExecContext(t.Context(), "INSERT INTO sys_user (username,password_hash,status) VALUES (?,'x',1)", username)
	if err != nil {
		t.Fatal(err)
	}
	userID, err = result.LastInsertId()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = db.ExecContext(t.Context(), "INSERT INTO casbin_rule (ptype,v0,v1,v2) VALUES ('g',?,?,?)", fmt.Sprintf("user:%d", userID), code, "dept:1"); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(t.Context(), "SELECT 1").Err(); err != nil {
		t.Fatal(err)
	}
	if err := roleModel.TransactCtx(t.Context(), func(ctx context.Context, s sqlx.Session) error { return rebuildRoleMenuPolicies(ctx, s, role.Id) }); err != nil {
		t.Fatal(err)
	}
	var count int64
	if err := conn.QueryRowCtx(t.Context(), &count, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0=? AND v1='dept:1' AND v2='/v1/sys/user/create' AND v3='POST' AND v4=''", code); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("mapped API policy not generated: %d", count)
	}
	if _, err := db.ExecContext(t.Context(), "DELETE FROM sys_role_menu WHERE role_id=?", roleID); err != nil {
		t.Fatal(err)
	}
	if err := roleModel.TransactCtx(t.Context(), func(ctx context.Context, s sqlx.Session) error { return rebuildRoleMenuPolicies(ctx, s, role.Id) }); err != nil {
		t.Fatal(err)
	}
	if err := conn.QueryRowCtx(t.Context(), &count, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0=?", code); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("policy not revoked: %d", count)
	}
	if err := conn.QueryRowCtx(t.Context(), &count, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='super_admin' AND v1='*' AND v2='/*' AND v3='*'"); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("super admin wildcard changed: %d", count)
	}
}

func TestReplaceMenuPermissionsRejectsUnknownAPI(t *testing.T) {
	_, conn := permissionTestDB(t)
	m := &customSysPermissionModel{defaultSysPermissionModel: &defaultSysPermissionModel{CachedConn: sqlc.NewConnWithCache(conn, permissionTestCache{}), table: "`sys_permission`"}}
	if err := m.ReplaceForMenu(t.Context(), 9001302, []int64{999999999}); err == nil {
		t.Fatal("unknown API id must be rejected")
	}
}

func TestCanDelegateScope(t *testing.T) {
	_, conn := permissionTestDB(t)
	if allowed, err := CanDelegateScope(t.Context(), conn.QueryRowsCtx, UserActor{UserId: 4, DeptId: 3}, "ALL"); err != nil || allowed {
		t.Fatalf("DEPT actor must not grant ALL: %v %v", allowed, err)
	}
	if allowed, err := CanDelegateScope(t.Context(), conn.QueryRowsCtx, UserActor{UserId: 4, DeptId: 3}, "DEPT"); err != nil || !allowed {
		t.Fatalf("DEPT actor must grant DEPT: %v %v", allowed, err)
	}
	if allowed, err := CanDelegateScope(t.Context(), conn.QueryRowsCtx, UserActor{UserId: 3, DeptId: 2}, "DEPT_AND_CHILDREN"); err != nil || !allowed {
		t.Fatalf("tree actor must grant descendants: %v %v", allowed, err)
	}
}
