package db_test

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"

	drivermysql "github.com/go-sql-driver/mysql"
	"go-zero-admin/pkg/testmysql"
	"gorm.io/driver/mysql"
)

const appFreshMarker = "-- App module: fresh installation includes administrator role 1 bootstrap."

func appSQL(t *testing.T, path string) string {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.ReplaceAll(string(source), "\r\n", "\n")
}

func TestAppFreshSQLContainsExactMigrationAndBootstrap(t *testing.T) {
	fresh := appSQL(t, "gozero-admin.sql")
	_, tail, found := strings.Cut(fresh, appFreshMarker+"\n")
	if !found || strings.Count(fresh, appFreshMarker) != 1 {
		t.Fatal("fresh installation must contain one App module tail")
	}
	migration := appSQL(t, "migrations/20261019_app_members.sql")
	bootstrap := appSQL(t, "operator/20261019_app_member_admin_bootstrap.sql")
	registration := appSQL(t, "migrations/20261019_app_registration_ip.sql")
	avatar := appSQL(t, "migrations/20261020_rustfs_avatar.sql")
	references := appSQL(t, "migrations/20261020_user_avatar_references.sql")
	avatarBootstrap := appSQL(t, "operator/20261020_avatar_admin_bootstrap.sql")
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(migration)))
	registrationChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(registration)))
	avatarChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(avatar)))
	referencesChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(references)))
	expected := migration + "\n" + bootstrap + "\n" +
		"INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('20261019_app_members.sql','" + checksum + "',NOW());\n\n" +
		registration + "\n" +
		"INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('20261019_app_registration_ip.sql','" + registrationChecksum + "',NOW());\n\n" +
		avatar + "\n" +
		"INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('20261020_rustfs_avatar.sql','" + avatarChecksum + "',NOW());\n\n" +
		references + "\n" + avatarBootstrap + "\n" +
		"INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('20261020_user_avatar_references.sql','" + referencesChecksum + "',NOW());\n"
	seed, found := strings.CutPrefix(tail, expected+"\n-- Fresh-install demo seed; never run against an existing database.\n")
	expectedSeed := appSQL(t, "seeds/demo.sql") + "\n" + appSQL(t, "seeds/activity.sql")
	if !found || strings.TrimSpace(seed) != strings.TrimSpace(expectedSeed) {
		t.Fatal("fresh installation differs from the standalone App SQL files and fresh-only demo seed")
	}
	if checksum != "d6b4f5ba9cdad5192f4234486d3bbe28e81cefde16c001601fb16a101f14bf50" {
		t.Fatal("already applied App membership migration must remain unchanged")
	}
}

func appMySQL(t *testing.T, source string) (*sql.DB, string) {
	t.Helper()
	db := testmysql.Open(t)
	dialector, ok := db.Dialector.(*mysql.Dialector)
	if !ok {
		t.Fatal("expected isolated MySQL fixture")
	}
	config, err := drivermysql.ParseDSN(dialector.DSN)
	if err != nil || !strings.HasPrefix(config.DBName, "gozero_test_rr_") {
		t.Fatal("App SQL tests require an isolated test database")
	}
	config.MultiStatements = true
	client, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal("cannot open isolated test connection")
	}
	t.Cleanup(func() { _ = client.Close() })
	if _, err := client.Exec(source); err != nil {
		t.Fatalf("isolated App SQL execution failed: %v", err)
	}
	return client, config.DBName
}

func appCount(t *testing.T, db *sql.DB, query string, want int, args ...any) {
	t.Helper()
	var got int
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("query %q: got %d, want %d", query, got, want)
	}
}

func assertAppSchema(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	for _, table := range []string{"app_users", "app_user_identities", "app_sessions", "app_membership_plans", "app_membership_grants"} {
		appCount(t, db, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name=?", 1, name, table)
	}
	appCount(t, db, "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND LEFT(table_name,4)='app_'", 5, name)
	appCount(t, db, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='app_user_identities' AND column_name='verified_at' AND is_nullable='YES' AND data_type='datetime'", 1, name)
	appCount(t, db, "SELECT COUNT(*) FROM app_membership_plans WHERE code='vip' AND version=1 AND enabled=1 AND JSON_CONTAINS(entitlements, JSON_QUOTE('profile.export'))", 1)
	appCount(t, db, "SELECT COUNT(*) FROM sys_base_menus WHERE name IN('app-members-users','app-memberships') AND deleted_at IS NULL", 2)
	appCount(t, db, "SELECT COUNT(*) FROM sys_base_menu_btns WHERE permission_key IN('app-members-users:review','app-members-users:status','app-members-users:revoke-session','app-memberships:grant','app-memberships:revoke') AND deleted_at IS NULL", 5)
	appCount(t, db, "SELECT COUNT(*) FROM sys_apis WHERE api_group='appmembers' AND deleted_at IS NULL", 9)
}

func assertAppAuthorization(t *testing.T, db *sql.DB, menus, buttons, apis int) {
	t.Helper()
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_menus a JOIN sys_base_menus m ON a.sys_base_menu_id=m.id WHERE a.sys_authority_authority_id=1 AND m.name IN('app-members-users','app-memberships')", menus)
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_btns a JOIN sys_base_menu_btns b ON a.sys_base_menu_btn_id=b.id WHERE a.authority_id=1 AND b.permission_key LIKE 'app-members%'", buttons)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v0='1' AND v1 LIKE '/v1/sys/app-members/%'", apis)
}

func assertAppRegistrationSchema(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	for column, length := range map[string]int{"registration_ip": 64, "registration_region": 255} {
		appCount(t, db, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='app_users' AND column_name=? AND data_type='varchar' AND character_maximum_length=? AND is_nullable='NO' AND column_default=''", 1, name, column, length)
	}
}
