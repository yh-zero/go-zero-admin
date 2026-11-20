package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlc"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

func TestRoleMenuConcurrentSavePreservesWinner(t *testing.T) {
	dsn := os.Getenv("GOZERO_ADMIN_TEST_DSN")
	if dsn == "" {
		t.Skip("requires isolated GOZERO_ADMIN_TEST_DSN with admin schema")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal("无法连接隔离测试数据库")
	}
	t.Cleanup(func() { _ = db.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)

	insert := func(t *testing.T, query string, args ...any) int64 {
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
			clean, stop := context.WithTimeout(context.Background(), 5*time.Second)
			defer stop()
			if _, err := db.ExecContext(clean, query, args...); err != nil {
				t.Errorf("清理本轮角色授权数据失败: %v", err)
			}
		})
	}
	code := fmt.Sprintf("role_cas_%d", time.Now().UnixNano())
	roleID := insert(t, "INSERT INTO sys_role(name,code) VALUES (?,?)", code, code)
	cleanup("DELETE FROM sys_role WHERE id=? AND code=?", roleID, code)
	menuIDs := make([]int64, 2)
	for i := range menuIDs {
		menuIDs[i] = insert(t, "INSERT INTO sys_menu(title,type) VALUES (?,2)", code)
		cleanup("DELETE FROM sys_menu WHERE id=? AND title=?", menuIDs[i], code)
	}
	cleanup("DELETE FROM sys_role_menu WHERE role_id=?", roleID)
	conn := sqlx.NewSqlConnFromDB(db)
	roles := &customSysRoleModel{&defaultSysRoleModel{CachedConn: sqlc.NewConnWithCache(conn, nil)}}
	initial, err := roles.ListMenuIdsByRole(ctx, roleID)
	if err != nil || len(initial) != 0 {
		t.Fatalf("新角色应无授权: %v, %v", initial, err)
	}

	// 两人读到同一空集合，使用各自的事务同时提交不同目标。
	type result struct {
		target int64
		err    error
	}
	start := make(chan struct{})
	results := make(chan result, len(menuIDs))
	for _, id := range menuIDs {
		go func(target int64) {
			<-start
			results <- result{target, roles.ReplaceMenus(ctx, uint64(roleID), initial, []int64{target})}
		}(id)
	}
	close(start)
	var winner int64
	conflicts := 0
	for range menuIDs {
		r := <-results
		switch {
		case r.err == nil:
			winner = r.target
		case errors.Is(r.err, ErrRoleMenuConflict):
			conflicts++
		default:
			t.Errorf("保存授权返回意外错误: %v", r.err)
		}
	}
	if winner == 0 || conflicts != 1 {
		t.Fatalf("同一基线应一成功一冲突，成功菜单=%d，冲突数=%d", winner, conflicts)
	}

	// 普通资料编辑携带过期的授权时，资料和授权都不能写入。
	newName := "冲突时不应写入"
	err = roles.UpdatePartial(ctx, uint64(roleID), RolePatch{Name: &newName, UpdateMenus: true, ExpectedMenuIds: initial})
	if !errors.Is(err, ErrRoleMenuConflict) {
		t.Fatalf("过期的角色编辑应报告冲突: %v", err)
	}
	var name string
	if err := db.QueryRowContext(ctx, "SELECT name FROM sys_role WHERE id=?", roleID).Scan(&name); err != nil || name != code {
		t.Fatalf("冲突后角色资料应保持原值: %q，%v", name, err)
	}
	current, err := roles.ListMenuIdsByRole(ctx, roleID)
	if err != nil || !slices.Equal(current, []int64{winner}) {
		t.Fatalf("先保存的授权应保留: %v，%v", current, err)
	}
	if err := roles.ReplaceMenus(ctx, uint64(roleID), current, menuIDs); err != nil {
		t.Fatalf("重读后应可提交: %v", err)
	}
	if err := roles.ReplaceMenus(ctx, uint64(roleID), []int64{menuIDs[1], menuIDs[0], menuIDs[0]}, nil); err != nil {
		t.Fatalf("原集合顺序或重复项不应影响清空授权: %v", err)
	}

	t.Run("核对后原菜单被删除仍应冲突", func(t *testing.T) {
		removed := insert(t, "INSERT INTO sys_menu(title,type) VALUES (?,2)", code)
		cleanup("DELETE FROM sys_menu WHERE id=? AND title=?", removed, code)
		if err := roles.ReplaceMenus(ctx, uint64(roleID), nil, []int64{removed}); err != nil {
			t.Fatal(err)
		}
		err := conn.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
			if _, err := lockRole(ctx, tx, uint64(roleID)); err != nil {
				return err
			}
			session := snapshotHookSession{tx, func(ctx context.Context) error {
				return conn.TransactCtx(ctx, func(ctx context.Context, deletion sqlx.Session) error {
					if _, err := deletion.ExecCtx(ctx, "UPDATE sys_menu SET deleted_at=NOW() WHERE id=?", removed); err != nil {
						return err
					}
					_, err := deletion.ExecCtx(ctx, "DELETE FROM sys_role_menu WHERE menu_id=?", removed)
					return err
				})
			}}
			return replaceRoleMenusIfCurrent(ctx, session, uint64(roleID), []int64{removed}, menuIDs)
		})
		if !errors.Is(err, ErrRoleMenuConflict) {
			t.Fatalf("菜单删除后的旧授权应报告冲突: %v", err)
		}
	})

	t.Run("不同角色可同时交换各自的菜单", func(t *testing.T) {
		secondCode := code + "_second"
		secondRole := insert(t, "INSERT INTO sys_role(name,code) VALUES (?,?)", secondCode, secondCode)
		cleanup("DELETE FROM sys_role WHERE id=? AND code=?", secondRole, secondCode)
		cleanup("DELETE FROM sys_role_menu WHERE role_id=?", secondRole)
		if _, err := db.ExecContext(ctx, "INSERT INTO sys_role_menu(role_id,menu_id) VALUES (?,?),(?,?)",
			roleID, menuIDs[0], secondRole, menuIDs[1]); err != nil {
			t.Fatal(err)
		}
		ready := make(chan struct{}, 2)
		proceed := make(chan struct{})
		completed := make(chan error, 2)
		swap := func(id, oldMenu, newMenu int64) {
			completed <- conn.TransactCtx(ctx, func(ctx context.Context, tx sqlx.Session) error {
				if _, err := lockRole(ctx, tx, uint64(id)); err != nil {
					return err
				}
				session := snapshotHookSession{tx, func(ctx context.Context) error {
					ready <- struct{}{}
					select {
					case <-proceed:
						return nil
					case <-ctx.Done():
						return ctx.Err()
					}
				}}
				return replaceRoleMenusIfCurrent(ctx, session, uint64(id), []int64{oldMenu}, []int64{newMenu})
			})
		}
		go swap(roleID, menuIDs[0], menuIDs[1])
		go swap(secondRole, menuIDs[1], menuIDs[0])
		for range 2 {
			select {
			case <-ready:
			case <-ctx.Done():
				t.Fatal("两个事务未能独立读到各自的初始授权")
			}
		}
		close(proceed)
		for range 2 {
			if err := <-completed; err != nil {
				t.Errorf("无角色授权冲突的保存应成功: %v", err)
			}
		}
		for id, want := range map[int64]int64{roleID: menuIDs[1], secondRole: menuIDs[0]} {
			got, err := roles.ListMenuIdsByRole(ctx, id)
			if err != nil || !slices.Equal(got, []int64{want}) {
				t.Errorf("角色 %d 的最终菜单应为 %d，得到 %v，%v", id, want, got, err)
			}
		}
	})
}

// 将并发的另一笔事务精确安排在读取快照与锁定菜单之间。
type snapshotHookSession struct {
	sqlx.Session
	afterSnapshot func(context.Context) error
}

func (s snapshotHookSession) QueryRowsCtx(ctx context.Context, rows any, query string, args ...any) error {
	if err := s.Session.QueryRowsCtx(ctx, rows, query, args...); err != nil {
		return err
	}
	if strings.HasPrefix(query, roleMenuIdsQuery) {
		return s.afterSnapshot(ctx)
	}
	return nil
}
