package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestDepartmentDirectoryRejectsMissingActorOrDepartment(t *testing.T) {
	departments := &customSysDeptModel{}
	for _, actor := range []UserActor{{DeptId: 1}, {UserId: 1}, {UserId: 1, DeptId: -1}} {
		_, err := departments.ListDepartmentDirectory(context.Background(), actor, "", 1, 20)
		if !errors.Is(err, ErrDataScopeDenied) {
			t.Fatalf("无有效用户或部门时应拒绝，实际错误 %v", err)
		}
	}
}

func TestDepartmentDirectoryRejectsUnboundedSearch(t *testing.T) {
	departments := &customSysDeptModel{}
	for _, test := range []struct {
		keyword        string
		page, pageSize int64
	}{
		{keyword: strings.Repeat("名", 65), page: 1, pageSize: 20},
		{page: 100001, pageSize: 20},
		{page: 1, pageSize: 101},
	} {
		_, err := departments.ListDepartmentDirectory(context.Background(), UserActor{UserId: 1, DeptId: 1}, test.keyword, test.page, test.pageSize)
		if !errors.Is(err, ErrInvalidMutation) {
			t.Fatalf("超出请求限制时应拒绝，实际错误 %v", err)
		}
	}
}

func TestDepartmentDirectoryVisibilityAndSearch(t *testing.T) {
	dsn := os.Getenv("GOZERO_ADMIN_TEST_DSN")
	if dsn == "" {
		t.Skip("需要配置带后台表结构的隔离 GOZERO_ADMIN_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("无法连接隔离测试数据库")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, stop := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(stop)

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
			clean, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if _, err := db.ExecContext(clean, query, args...); err != nil {
				t.Errorf("清理本轮通讯录测试数据失败: %v", err)
			}
		})
	}
	prefix := fmt.Sprintf("dir_%d", time.Now().UnixNano())
	createDepartment := func(suffix string, parent int64) int64 {
		name := prefix + suffix
		id := insert("INSERT INTO sys_dept (parent_id,name,path) VALUES (?,?,?)", parent, name, "/")
		cleanup("DELETE FROM sys_dept WHERE id=? AND name=?", id, name)
		return id
	}
	createUser := func(suffix, name, nickname string, deptId int64, status int, deletedAt any) int64 {
		username := prefix + suffix
		id := insert(`INSERT INTO sys_user (username,password_hash,real_name,nickname,email,dept_id,status,deleted_at)
VALUES (?,'test-hash',?,?,'directory@example.test',?,?,?)`, username, name, nickname, deptId, status, deletedAt)
		cleanup("DELETE FROM sys_user WHERE id=? AND username=?", id, username)
		return id
	}
	mainDept := createDepartment("_main", 0)
	childDept := createDepartment("_child", mainDept)
	viewer := createUser("_viewer", "", "", mainDept, 1, nil)
	leader := createUser("_leader", "林%明", "另一个昵称", mainDept, 1, nil)
	createUser("_disabled", "停用者", "", mainDept, 2, nil)
	createUser("_deleted", "已删除", "", mainDept, 1, time.Now())
	createUser("_child_user", "下级成员", "", childDept, 1, nil)
	if _, err := db.ExecContext(ctx, "UPDATE sys_dept SET leader_user_id=? WHERE id=?", leader, mainDept); err != nil {
		t.Fatal(err)
	}
	departments := &customSysDeptModel{&defaultSysDeptModel{
		CachedConn: sqlc.NewConnWithCache(sqlx.NewSqlConnFromDB(db), nil),
	}}
	actor := UserActor{UserId: viewer, DeptId: mainDept}
	first, err := departments.ListDepartmentDirectory(ctx, actor, "", 1, 1)
	if err != nil || first.DeptName != prefix+"_main" || first.Total != 2 || len(first.List) != 1 || first.List[0].Id != leader || first.List[0].Name != "林%明" || !first.List[0].IsLeader {
		t.Fatalf("第一页应优先显示本部门负责人，统计只包含启用直属成员: %+v %v", first, err)
	}
	second, err := departments.ListDepartmentDirectory(ctx, actor, "", 2, 1)
	if err != nil || len(second.List) != 1 || second.List[0].Id != viewer || second.List[0].Name != "未设置姓名" || second.List[0].IsLeader {
		t.Fatalf("第二页应保留无姓名成员且隐去登录名: %+v %v", second, err)
	}
	filtered, err := departments.ListDepartmentDirectory(ctx, actor, "%", 1, 20)
	if err != nil || filtered.Total != 1 || len(filtered.List) != 1 || filtered.List[0].Id != leader {
		t.Fatalf("百分号应按字面搜索: %+v %v", filtered, err)
	}
	forged := UserActor{UserId: viewer, DeptId: childDept}
	if _, err = departments.ListDepartmentDirectory(ctx, forged, "", 1, 20); !errors.Is(err, ErrDataScopeDenied) {
		t.Fatalf("未经独立授权的下级部门应拒绝: %v", err)
	}
	roleCode := prefix + "_role"
	roleId := insert("INSERT INTO sys_role(name,code) VALUES (?,?)", roleCode, roleCode)
	cleanup("DELETE FROM sys_role WHERE id=? AND code=?", roleId, roleCode)
	policyId := insert("INSERT INTO casbin_rule(ptype,v0,v1,v2) VALUES ('g',?,?,?)", fmt.Sprintf("user:%d", viewer), roleCode, fmt.Sprintf("dept:%d", childDept))
	cleanup("DELETE FROM casbin_rule WHERE id=? AND v1=?", policyId, roleCode)
	child, err := departments.ListDepartmentDirectory(ctx, forged, "", 1, 20)
	if err != nil || child.Total != 1 || len(child.List) != 1 || child.List[0].Name != "下级成员" {
		t.Fatalf("身份有有效部门角色时应只查看该部门: %+v %v", child, err)
	}
	if _, err := db.ExecContext(ctx, "UPDATE sys_role SET status=2 WHERE id=?", roleId); err != nil {
		t.Fatal(err)
	}
	if _, err := departments.ListDepartmentDirectory(ctx, forged, "", 1, 20); !errors.Is(err, ErrDataScopeDenied) {
		t.Fatalf("停用的角色不能继续访问该部门: %v", err)
	}
}
