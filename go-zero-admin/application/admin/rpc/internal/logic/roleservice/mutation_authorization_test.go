package roleservicelogic

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/casbinauth"
)

type roleMutationTestCache struct{ cache.Cache }

func (roleMutationTestCache) DelCtx(context.Context, ...string) error { return nil }

func TestRequireRoleMutationAllowed(t *testing.T) {
	dsn := os.Getenv("GOZERO_ADMIN_TEST_DSN")
	if dsn == "" {
		t.Skip("需要隔离的 GOZERO_ADMIN_TEST_DSN")
	}
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	conn := sqlx.NewSqlConnFromDB(db)
	cacheConf := cache.CacheConf{{RedisConf: redis.RedisConf{Host: "127.0.0.1:16379", Type: "node"}, Weight: 100}}
	svcCtx := &svc.ServiceContext{DB: conn, SysUserModel: model.NewSysUserModel(conn, cacheConf), SysRoleModel: model.NewSysRoleModel(conn, cacheConf), SysPermissionModel: model.NewSysPermissionModel(conn, cacheConf), CasbinRuleModel: model.NewCasbinRuleModel(conn, cacheConf)}
	rows, err := svcCtx.CasbinRuleModel.ListActivePolicies(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	rules := make([]casbinauth.PolicyRule, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, casbinauth.PolicyRule{Ptype: row.Ptype, V0: row.V0, V1: row.V1, V2: row.V2, V3: row.V3, V4: row.V4, V5: row.V5})
	}
	svcCtx.Enforcer, err = casbinauth.NewHolder(rules)
	if err != nil {
		t.Fatal(err)
	}
	admin := &adminpb.Actor{UserId: 1, DeptId: 1}
	if err := requireRoleMutationAllowed(t.Context(), svcCtx, admin, roleMutationInput{RoleID: 4}); err != nil {
		t.Fatalf("super admin should manage role: %v", err)
	}
	actor := &adminpb.Actor{UserId: 4, DeptId: 3}
	if err := requireRoleMutationAllowed(t.Context(), svcCtx, actor, roleMutationInput{RoleID: 4}); err == nil {
		t.Fatal("self-bound role mutation must be denied")
	}
	if err := requireRoleMutationAllowed(t.Context(), svcCtx, actor, roleMutationInput{Code: "delegated_test", DataScope: "ALL"}); err == nil {
		t.Fatal("scope escalation must be denied")
	}
	if err := requireRoleMutationAllowed(t.Context(), svcCtx, actor, roleMutationInput{Code: "delegated_test", DataScope: "SELF"}); err != nil {
		t.Fatalf("delegated role within own scope should be allowed: %v", err)
	}
}
