package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"golang.org/x/crypto/bcrypt"
)

type importCacheRecorder struct {
	cache.Cache
	mu   sync.Mutex
	keys []string
}

func (c *importCacheRecorder) DelCtx(_ context.Context, keys ...string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.keys = append(c.keys, keys...)
	return nil
}

func TestUserImportRespectsScopeAndDatabaseUniqueness(t *testing.T) {
	dsn := os.Getenv("GOZERO_ADMIN_TEST_DSN")
	if dsn == "" {
		t.Skip("需要配置隔离的 GOZERO_ADMIN_TEST_DSN")
	}
	sqlx.DisableLog()
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("无法连接隔离测试数据库")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	t.Cleanup(cancel)
	insert := func(query string, args ...any) int64 {
		t.Helper()
		result, err := db.ExecContext(ctx, query, args...)
		if err != nil {
			t.Fatal(err)
		}
		id, err := result.LastInsertId()
		if err != nil {
			t.Fatal(err)
		}
		return id
	}
	cleanup := func(query string, args ...any) {
		t.Cleanup(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := db.ExecContext(ctx, query, args...); err != nil {
				t.Errorf("清理导入测试数据失败: %v", err)
			}
		})
	}
	prefix := fmt.Sprintf("ui%d", time.Now().UnixNano())
	home := insert("INSERT INTO sys_dept (name,path) VALUES (?,'/')", prefix+"home")
	cleanup("DELETE FROM sys_dept WHERE id=? AND name=?", home, prefix+"home")
	outside := insert("INSERT INTO sys_dept (name,path) VALUES (?,'/')", prefix+"outside")
	cleanup("DELETE FROM sys_dept WHERE id=? AND name=?", outside, prefix+"outside")
	adminId := insert("INSERT INTO sys_user (username,password_hash,dept_id) VALUES (?,'test',?)", prefix+"actor", home)
	cleanup("DELETE FROM sys_user WHERE id=? AND username=?", adminId, prefix+"actor")
	roleId := insert("INSERT INTO sys_role (name,code,data_scope) VALUES (?,?,'DEPT')", prefix, prefix)
	cleanup("DELETE FROM sys_role WHERE id=? AND code=?", roleId, prefix)
	policyId := insert("INSERT INTO casbin_rule (ptype,v0,v1,v2) VALUES ('g',?,?,?)", fmt.Sprintf("user:%d", adminId), prefix, fmt.Sprintf("dept:%d", home))
	cleanup("DELETE FROM casbin_rule WHERE id=? AND v1=?", policyId, prefix)
	reserved := insert("INSERT INTO sys_user (username,password_hash,dept_id,deleted_at) VALUES (?,'test',?,NOW())", prefix+"reserved", outside)
	cleanup("DELETE FROM sys_user WHERE id=? AND username=?", reserved, prefix+"reserved")
	cleanup("DELETE FROM sys_user WHERE username=? AND dept_id=?", prefix+"first", home)
	userCache := &importCacheRecorder{}
	users := &customSysUserModel{&defaultSysUserModel{CachedConn: sqlc.NewConnWithCache(sqlx.NewSqlConnFromDB(db), userCache)}}
	actor := UserActor{UserId: adminId, DeptId: home}
	rows := []UserImportCandidate{
		{Row: 2, Username: prefix + "first", DeptId: home, RealName: "测试成员"},
		{Row: 3, Username: prefix + "unchecked", DeptId: home},
		{Row: 4, Username: prefix + "outside", DeptId: outside},
		{Row: 5, Username: prefix + "RESERVED", DeptId: home},
		{Row: 6, Username: prefix + "Cafe", DeptId: home},
		{Row: 7, Username: prefix + "Café", DeptId: home},
	}
	cleanup("DELETE FROM sys_user WHERE username=? AND dept_id=?", prefix+"unchecked", home)
	cleanup("DELETE FROM sys_user WHERE username=? AND dept_id=?", prefix+"outside", outside)
	cleanup("DELETE FROM sys_user WHERE username=? AND dept_id=?", prefix+"Cafe", home)
	issues, err := users.ImportUsers(ctx, actor, rows, nil)
	if err != nil || issues[2] != nil || issues[3] != nil || !errors.Is(issues[4], ErrDataScopeDenied) ||
		!errors.Is(issues[5], ErrUsernameExists) || !errors.Is(issues[6], ErrImportDuplicate) || !errors.Is(issues[7], ErrImportDuplicate) {
		t.Fatalf("预检应区分可导入、越界、保留账号和同表重名: %v，%v", issues, err)
	}
	var count int
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sys_user WHERE username=?", prefix+"first").Scan(&count); err != nil || count != 0 {
		t.Fatalf("预检不能创建账号: %d, %v", count, err)
	}
	issues, err = users.ImportUsers(ctx, actor, rows, []int64{2, 4, 6})
	if err != nil || issues[2] != nil || !errors.Is(issues[4], ErrDataScopeDenied) || !errors.Is(issues[6], ErrImportDuplicate) {
		t.Fatalf("确认需重新校验选中的记录以及整个文件的重复账号: %v，%v", issues, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM sys_user WHERE username IN (?, ?, ?)", prefix+"unchecked", prefix+"outside", prefix+"Cafe").Scan(&count); err != nil || count != 0 {
		t.Fatalf("只允许写入选中的合法记录: %d, %v", count, err)
	}
	var hash, name string
	var userId uint64
	var status int
	if err := db.QueryRowContext(ctx, "SELECT id,password_hash,real_name,status FROM sys_user WHERE username=?", prefix+"first").Scan(&userId, &hash, &name, &status); err != nil || status != 2 || name != "测试成员" || !errors.Is(bcrypt.CompareHashAndPassword([]byte(hash), []byte("SomePass123")), bcrypt.ErrHashTooShort) {
		t.Fatalf("导入账号应保留姓名并停用，密码必须重置后才能登录: %s %d %v", name, status, err)
	}
	if expected := []string{fmt.Sprintf("%s%d", cacheGozeroAdminNewSysUserIdPrefix, userId), cacheGozeroAdminNewSysUserUsernamePrefix + prefix + "first"}; !slices.Equal(userCache.keys, expected) {
		t.Fatalf("创建应在提交后失效账号与 ID 缓存: %v", userCache.keys)
	}
	issues, err = users.ImportUsers(ctx, actor, []UserImportCandidate{{Row: 2, Username: prefix + "FIRST", DeptId: home, RealName: "覆盖失败"}}, []int64{2})
	if err != nil || !errors.Is(issues[2], ErrUsernameExists) {
		t.Fatalf("再次提交不能覆盖已创建账号: %v，%v", issues, err)
	}
	if err := db.QueryRowContext(ctx, "SELECT real_name FROM sys_user WHERE username=?", prefix+"first").Scan(&name); err != nil || name != "测试成员" {
		t.Fatalf("原账号资料应保留: %s, %v", name, err)
	}
	enabled := int32(1)
	if err := users.UpdatePartial(ctx, int64(userId), actor, UserPatch{Status: &enabled}); err != nil {
		t.Fatalf("管理员应可通过现有编辑路径启用导入账号: %v", err)
	}
	user, err := users.FindScopedUser(ctx, int64(userId), actor)
	if err != nil || user.Status != int64(enabled) || user.PasswordHash != hash {
		t.Fatalf("启用状态不得自动建立登录凭据: %v", err)
	}
	password := []byte("Directory!Test123")
	newHash, err := bcrypt.GenerateFromPassword(password, bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	if err := users.UpdatePassword(ctx, userId, actor, "", string(newHash), func(_ context.Context, id int64) error {
		if id != int64(userId) {
			t.Errorf("应撤销导入账户的会话，实际用户 ID %d", id)
		}
		return nil
	}); err != nil {
		t.Fatalf("启用后应可通过现有密码重置路径建立登录凭据: %v", err)
	}
	if err := db.QueryRowContext(ctx, "SELECT password_hash FROM sys_user WHERE id=?", userId).Scan(&hash); err != nil || bcrypt.CompareHashAndPassword([]byte(hash), password) != nil {
		t.Fatalf("密码重置应替换导入占位值: %v", err)
	}

	raceRows := []UserImportCandidate{{Row: 2, Username: prefix + "race", DeptId: home}}
	cleanup("DELETE FROM sys_user WHERE username=? AND dept_id=?", raceRows[0].Username, home)
	start := make(chan struct{})
	results := make(chan error, 2)
	for range 2 {
		go func() {
			<-start
			issues, err := users.ImportUsers(ctx, actor, raceRows, []int64{2})
			if err == nil {
				err = issues[2]
			}
			results <- err
		}()
	}
	close(start)
	created, duplicates := 0, 0
	for range 2 {
		switch err := <-results; {
		case err == nil:
			created++
		case errors.Is(err, ErrUsernameExists), errors.Is(err, ErrImportConflict):
			duplicates++
		default:
			t.Errorf("并发确认失败原因异常: %v", err)
		}
	}
	if created != 1 || duplicates != 1 {
		t.Fatalf("并发确认应只创建一次并报告重复: %d 次创建，%d 次重复", created, duplicates)
	}
}
