package db_test

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"go-zero-admin/pkg/hash"
)

func TestPermissionKeyCapacityPreservesHistoricalNames(t *testing.T) {
	fresh, err := os.ReadFile("gozero-admin.sql")
	if err != nil {
		t.Fatal(err)
	}
	nameCapacity := func(table string) int {
		t.Helper()
		block := regexp.MustCompile("(?s)CREATE TABLE `" + table + "` \\((.*?)\\) ENGINE").FindSubmatch(fresh)
		if len(block) != 2 {
			t.Fatalf("missing historical table definition: %s", table)
		}
		length := regexp.MustCompile("`name` varchar\\(([0-9]+)\\)").FindSubmatch(block[1])
		if len(length) != 2 {
			t.Fatalf("missing historical name length: %s", table)
		}
		n, err := strconv.Atoi(string(length[1]))
		if err != nil {
			t.Fatal(err)
		}
		return n
	}
	minimum := nameCapacity("sys_base_menus") + 1 + nameCapacity("sys_base_menu_btns")
	for _, path := range []string{"gozero-admin.sql", "migrations/20261008_permission_revision_history.sql"} {
		source, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		lengths := regexp.MustCompile("permission_key varchar\\(([0-9]+)\\)").FindAllSubmatch(source, -1)
		if len(lengths) != 2 {
			t.Fatalf("%s: expected ADD and MODIFY key definitions", path)
		}
		for _, length := range lengths {
			capacity, _ := strconv.Atoi(string(length[1]))
			if capacity < minimum {
				t.Fatalf("%s: key capacity %d truncates valid historical names requiring %d", path, capacity, minimum)
			}
		}
		if !strings.Contains(string(source), "ALTER TABLE sys_base_menu_btns ROW_FORMAT=DYNAMIC") {
			t.Fatalf("%s: expanded utf8mb4 unique key requires DYNAMIC row format for historical REDUNDANT tables", path)
		}
	}
}

// Import the actual fresh-install SQL once in an isolated schema and check the
// permission and App portions together; this catches drift that AutoMigrate-only
// fixtures cannot. The former permission and App tests each imported the same
// file, so they shared no extra coverage.
func TestMySQLFreshSchema(t *testing.T) {
	db, name := appMySQL(t, appSQL(t, "gozero-admin.sql"))
	for _, table := range []string{"sys_permission_versions", "sys_permission_changes"} {
		appCount(t, db, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name=?", 1, name, table)
	}
	appCount(t, db, "SELECT COUNT(*) FROM sys_base_menu_btns WHERE permission_key IS NULL OR permission_key = ''", 0)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1='/v1/sys/permissions/edit' AND v2='GET'", 1)
	migration, err := os.ReadFile("migrations/20261008_permission_revision_history.sql")
	if err != nil {
		t.Fatal(err)
	}
	permission := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ReplaceAll(string(migration), "\r\n", "\n"))))
	appCount(t, db, "SELECT COUNT(*) FROM schema_migrations WHERE filename=? AND checksum=?", 1, "20261008_permission_revision_history.sql", permission)

	assertAppSchema(t, db, name)
	assertAppRegistrationSchema(t, db, name)
	assertAppAuthorization(t, db, 2, 6, 9)
	assertFreshDemoSeeds(t, db)
	for _, filename := range []string{"20261019_app_members.sql", "20261019_app_registration_ip.sql"} {
		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(appSQL(t, "migrations/"+filename))))
		appCount(t, db, "SELECT COUNT(*) FROM schema_migrations WHERE filename=? AND checksum=?", 1, filename, checksum)
	}
}

func assertFreshDemoSeeds(t *testing.T, db *sql.DB) {
	t.Helper()
	appCount(t, db, "SELECT COUNT(*) FROM sys_users", 4)
	appCount(t, db, "SELECT COUNT(*) FROM sys_authorities", 3)
	appCount(t, db, "SELECT COUNT(*) FROM sys_user_authority", 4)
	for username, role := range map[string]int{"admin": 1, "operator": 2, "viewer": 3, "disabled": 3} {
		enable := 1
		if username == "disabled" {
			enable = 2
		}
		appCount(t, db, `SELECT COUNT(*) FROM sys_users u JOIN sys_user_authority a ON a.sys_user_id=u.id
 WHERE u.username=? AND u.authority_id=? AND a.sys_authority_authority_id=? AND u.enable=? AND u.deleted_at IS NULL`, 1, username, role, role, enable)
		var encoded string
		if err := db.QueryRow("SELECT password FROM sys_users WHERE username=?", username).Scan(&encoded); err != nil {
			t.Fatal(err)
		}
		password := "Demo@123456"
		if username == "admin" {
			password = "123456"
		}
		if !hash.BcryptCheck(password, encoded) {
			t.Fatalf("fresh account %s does not match its documented password", username)
		}
	}
	appCount(t, db, "SELECT COUNT(*) FROM app_users", 6)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE status='active'", 3)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE status='pending'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE status='disabled'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE status='rejected'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE avatar_url<>'' OR registration_ip<>'' OR registration_region<>''", 0)
	appCount(t, db, "SELECT COUNT(*) FROM app_user_identities", 6)
	appCount(t, db, "SELECT COUNT(*) FROM app_user_identities WHERE kind='email' AND identifier LIKE '%@example.test'", 6)
	appCount(t, db, "SELECT COUNT(*) FROM app_user_identities WHERE verified_at IS NOT NULL", 0)
	rows, err := db.Query("SELECT identifier,password_hash FROM app_user_identities")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var identifier, encoded string
		if err := rows.Scan(&identifier, &encoded); err != nil {
			t.Fatal(err)
		}
		if !hash.BcryptCheck("Demo@123456", encoded) {
			t.Fatalf("fresh App account %s does not match its documented password", identifier)
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	appCount(t, db, "SELECT COUNT(*) FROM app_membership_grants", 3)
	appCount(t, db, `SELECT COUNT(*) FROM app_membership_grants g JOIN app_users u ON u.id=g.user_id
 JOIN app_membership_plans p ON p.id=g.plan_id
 WHERE u.status='active' AND p.code='vip' AND g.plan_code=p.code AND g.plan_version=p.version
 AND JSON_CONTAINS(g.entitlements,JSON_QUOTE('profile.export')) AND g.days=30
 AND TIMESTAMPDIFF(SECOND,g.starts_at,g.ends_at)=30*86400
 AND g.starts_at<=UTC_TIMESTAMP(3) AND g.ends_at>UTC_TIMESTAMP(3) AND g.revoked_at IS NULL`, 1)
	for _, table := range []string{
		"sys_device_sessions", "app_sessions", "sys_permission_changes",
	} {
		appCount(t, db, "SELECT COUNT(*) FROM "+table, 0)
	}
	appCount(t, db, "SELECT COUNT(*) FROM app_membership_grants WHERE ends_at<UTC_TIMESTAMP(3) AND revoked_at IS NULL", 1)
	appCount(t, db, "SELECT COUNT(*) FROM app_membership_grants WHERE revoked_at IS NOT NULL", 1)
	appCount(t, db, "SELECT COUNT(*) FROM app_membership_grants WHERE user_id=1 AND starts_at<=UTC_TIMESTAMP(3) AND ends_at>UTC_TIMESTAMP(3) AND revoked_at IS NULL", 0)
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_resources WHERE status='active'", 3)
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_resources WHERE visibility='public'", 2)
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_resources WHERE visibility='private'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM sys_file_references r JOIN sys_file_resources f ON f.id=r.file_id WHERE r.object_type='demo_attachment' AND f.visibility='public'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM sys_audit_logs WHERE action LIKE '内置演示：%'", 3)
	appCount(t, db, "SELECT COUNT(*) FROM sys_ai_conversations WHERE owner_id=1", 1)
	appCount(t, db, "SELECT COUNT(*) FROM sys_ai_messages WHERE owner_id=1", 2)
	appCount(t, db, "SELECT COUNT(*) FROM sys_ai_runs WHERE owner_id=1 AND status='succeeded' AND input_tokens=0 AND output_tokens=0 AND session_id='' AND lease_owner=''", 1)
	appCount(t, db, `SELECT COUNT(*) FROM (SELECT parent_id,sort FROM sys_base_menus
 WHERE deleted_at IS NULL GROUP BY parent_id,sort HAVING COUNT(*)>1) duplicates`, 0)
	assertFreshSeedRelations(t, db)
	assertFreshDemoPermissions(t, db)
}

func assertFreshSeedRelations(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, query := range []string{
		`SELECT COUNT(*) FROM sys_users u WHERE NOT EXISTS(SELECT 1 FROM sys_authorities a WHERE a.authority_id=u.authority_id AND a.deleted_at IS NULL)
 OR NOT EXISTS(SELECT 1 FROM sys_user_authority a WHERE a.sys_user_id=u.id AND a.sys_authority_authority_id=u.authority_id)`,
		`SELECT COUNT(*) FROM sys_user_authority a LEFT JOIN sys_users u ON u.id=a.sys_user_id LEFT JOIN sys_authorities r ON r.authority_id=a.sys_authority_authority_id WHERE u.id IS NULL OR r.authority_id IS NULL`,
		`SELECT COUNT(*) FROM sys_authorities r WHERE r.default_router<>'' AND NOT EXISTS(SELECT 1 FROM sys_base_menus m JOIN sys_authority_menus a ON a.sys_base_menu_id=m.id WHERE m.name=r.default_router AND m.deleted_at IS NULL AND a.sys_authority_authority_id=r.authority_id)`,
		`SELECT COUNT(*) FROM sys_base_menus m WHERE m.parent_id<>0 AND NOT EXISTS(SELECT 1 FROM sys_base_menus p WHERE p.id=m.parent_id AND p.deleted_at IS NULL)`,
		`SELECT COUNT(*) FROM sys_authorities r WHERE r.parent_id<>0 AND NOT EXISTS(SELECT 1 FROM sys_authorities p WHERE p.authority_id=r.parent_id AND p.deleted_at IS NULL)`,
		`SELECT COUNT(*) FROM sys_authority_menus a LEFT JOIN sys_base_menus m ON m.id=a.sys_base_menu_id LEFT JOIN sys_authorities r ON r.authority_id=a.sys_authority_authority_id WHERE m.id IS NULL OR r.authority_id IS NULL`,
		`SELECT COUNT(*) FROM sys_base_menu_btns b LEFT JOIN sys_base_menus m ON m.id=b.sys_base_menu_id WHERE m.id IS NULL OR b.permission_key<>CONCAT(m.name,':',b.name)`,
		`SELECT COUNT(*) FROM sys_authority_btns a LEFT JOIN sys_base_menu_btns b ON b.id=a.sys_base_menu_btn_id LEFT JOIN sys_authorities r ON r.authority_id=a.authority_id
 WHERE b.id IS NULL OR r.authority_id IS NULL OR b.sys_base_menu_id<>a.sys_menu_id OR NOT EXISTS(SELECT 1 FROM sys_authority_menus m WHERE m.sys_base_menu_id=a.sys_menu_id AND m.sys_authority_authority_id=a.authority_id)`,
		`SELECT COUNT(*) FROM casbin_rule p WHERE p.ptype<>'p' OR NOT EXISTS(SELECT 1 FROM sys_authorities r WHERE r.authority_id=CAST(p.v0 AS UNSIGNED))
 OR NOT EXISTS(SELECT 1 FROM sys_apis a WHERE a.path COLLATE utf8mb4_bin=p.v1 COLLATE utf8mb4_bin AND a.method COLLATE utf8mb4_bin=p.v2 COLLATE utf8mb4_bin AND a.deleted_at IS NULL)`,
		`SELECT COUNT(*) FROM app_users u WHERE NOT EXISTS(SELECT 1 FROM app_user_identities i WHERE i.user_id=u.id)`,
		`SELECT COUNT(*) FROM app_user_identities i LEFT JOIN app_users u ON u.id=i.user_id WHERE u.id IS NULL`,
		`SELECT COUNT(*) FROM app_membership_grants g LEFT JOIN app_users u ON u.id=g.user_id LEFT JOIN app_membership_plans p ON p.id=g.plan_id LEFT JOIN sys_users a ON a.id=g.granted_by WHERE u.id IS NULL OR p.id IS NULL OR a.id IS NULL`,
		`SELECT COUNT(*) FROM sys_departments d WHERE d.parent_id<>0 AND NOT EXISTS(SELECT 1 FROM sys_departments p WHERE p.id=d.parent_id AND p.deleted_at IS NULL)`,
		`SELECT COUNT(*) FROM sys_user_departments a LEFT JOIN sys_users u ON u.id=a.user_id LEFT JOIN sys_departments d ON d.id=a.department_id WHERE u.id IS NULL OR d.id IS NULL`,
		`SELECT COUNT(*) FROM sys_user_positions a LEFT JOIN sys_users u ON u.id=a.user_id LEFT JOIN sys_positions p ON p.id=a.position_id WHERE u.id IS NULL OR p.id IS NULL`,
		`SELECT COUNT(*) FROM sys_role_data_scopes a LEFT JOIN sys_authorities r ON r.authority_id=a.authority_id WHERE r.authority_id IS NULL`,
	} {
		appCount(t, db, query, 0)
	}
}

func assertFreshDemoPermissions(t *testing.T, db *sql.DB) {
	t.Helper()
	for _, query := range []string{
		`SELECT COUNT(*) FROM sys_base_menus m WHERE m.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM sys_authority_menus a WHERE a.sys_authority_authority_id=1 AND a.sys_base_menu_id=m.id)`,
		`SELECT COUNT(*) FROM sys_base_menu_btns b WHERE b.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM sys_authority_btns a WHERE a.authority_id=1 AND a.sys_base_menu_btn_id=b.id)`,
		`SELECT COUNT(*) FROM sys_apis a WHERE a.deleted_at IS NULL AND NOT EXISTS(SELECT 1 FROM casbin_rule p WHERE p.ptype='p' AND p.v0='1' AND p.v1 COLLATE utf8mb4_bin=a.path COLLATE utf8mb4_bin AND p.v2 COLLATE utf8mb4_bin=a.method COLLATE utf8mb4_bin)`,
	} {
		appCount(t, db, query, 0)
	}
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_menus WHERE sys_authority_authority_id=2", 6)
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_menus WHERE sys_authority_authority_id=3", 3)
	appCount(t, db, `SELECT COUNT(*) FROM sys_authority_menus a JOIN sys_base_menus m ON m.id=a.sys_base_menu_id
 WHERE a.sys_authority_authority_id=2 AND m.name IN('index','superAdmin','files','businessTools','app-members-users','app-memberships')`, 6)
	appCount(t, db, `SELECT COUNT(*) FROM sys_authority_menus a JOIN sys_base_menus m ON m.id=a.sys_base_menu_id
 WHERE a.sys_authority_authority_id=3 AND m.name IN('index','app-members-users','app-memberships')`, 3)
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_btns WHERE authority_id=2", 5)
	appCount(t, db, "SELECT COUNT(DISTINCT b.permission_key) FROM sys_authority_btns a JOIN sys_base_menu_btns b ON b.id=a.sys_base_menu_btn_id WHERE a.authority_id=2", 5)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='2'", 10)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='3'", 5)
	for _, role := range []string{"2", "3"} {
		for _, path := range []string{"/v1/sys/app-members/users", "/v1/sys/app-members/sessions"} {
			appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0=? AND v1=? AND v2='GET'", 1, role, path)
		}
	}
	for _, path := range []string{"/v1/sys/app-members/status", "/v1/sys/app-members/avatar"} {
		appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='2' AND v1=? AND v2='PUT'", 1, path)
	}
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='2' AND v1='/v1/sys/base/uploadFileImg' AND v2='POST'", 1)
	appCount(t, db, `SELECT COUNT(*) FROM casbin_rule WHERE v0='2' AND NOT (
 (v2='GET' AND v1 IN('/v1/sys/menu/getMenu','/v1/sys/app-members/users','/v1/sys/app-members/sessions','/v1/sys/app-members/plans','/v1/sys/app-members/grants','/v1/sys/files/list','/v1/sys/files/url'))
 OR (v2='PUT' AND v1 IN('/v1/sys/app-members/status','/v1/sys/app-members/avatar'))
 OR (v2='POST' AND v1='/v1/sys/base/uploadFileImg'))`, 0)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE v0='3' AND v2<>'GET'", 0)
	appCount(t, db, `SELECT COUNT(*) FROM casbin_rule WHERE v0='3' AND v1 NOT IN(
 '/v1/sys/menu/getMenu','/v1/sys/app-members/users','/v1/sys/app-members/sessions','/v1/sys/app-members/plans','/v1/sys/app-members/grants')`, 0)
	for _, path := range []string{"/v1/sys/app-members/plans", "/v1/sys/app-members/grants"} {
		appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='3' AND v1=? AND v2='GET'", 1, path)
	}
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_btns WHERE authority_id=3", 0)
	appCount(t, db, `SELECT COUNT(*) FROM sys_authority_btns a JOIN sys_base_menu_btns b ON b.id=a.sys_base_menu_btn_id
 WHERE a.authority_id=2 AND b.permission_key NOT IN('app-members-users:review','app-members-users:status','app-members-users:avatar','files:upload','businessTools:upload')`, 0)
	appCount(t, db, "SELECT COUNT(*) FROM sys_role_data_scopes WHERE authority_id IN(2,3) AND scope<>'self'", 0)
}
