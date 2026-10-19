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
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(migration)))
	registrationChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte(registration)))
	expected := migration + "\n" + bootstrap + "\n" +
		"INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('20261019_app_members.sql','" + checksum + "',NOW());\n\n" +
		registration + "\n" +
		"INSERT INTO schema_migrations(filename,checksum,applied_at) VALUES('20261019_app_registration_ip.sql','" + registrationChecksum + "',NOW());\n"
	if tail != expected {
		t.Fatal("fresh installation differs from the standalone App SQL files")
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
	appExec(t, client, source)
	return client, config.DBName
}

func appExec(t *testing.T, db *sql.DB, source string) {
	t.Helper()
	if _, err := db.Exec(source); err != nil {
		t.Fatalf("isolated App SQL execution failed: %v", err)
	}
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
	appCount(t, db, "SELECT COUNT(*) FROM sys_apis WHERE api_group='appmembers' AND deleted_at IS NULL", 8)
}

func assertAppAuthorization(t *testing.T, db *sql.DB, menus, buttons, apis int) {
	t.Helper()
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_menus a JOIN sys_base_menus m ON a.sys_base_menu_id=m.id WHERE m.name IN('app-members-users','app-memberships')", menus)
	appCount(t, db, "SELECT COUNT(*) FROM sys_authority_btns a JOIN sys_base_menu_btns b ON a.sys_base_menu_btn_id=b.id WHERE b.permission_key LIKE 'app-members%'", buttons)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v1 LIKE '/v1/sys/app-members/%'", apis)
}

func assertAppRegistrationSchema(t *testing.T, db *sql.DB, name string) {
	t.Helper()
	for column, length := range map[string]int{"registration_ip": 64, "registration_region": 255} {
		appCount(t, db, "SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='app_users' AND column_name=? AND data_type='varchar' AND character_maximum_length=? AND is_nullable='NO' AND column_default=''", 1, name, column, length)
	}
}

func TestMySQLFreshAppSchema(t *testing.T) {
	db, name := appMySQL(t, appSQL(t, "gozero-admin.sql"))
	assertAppSchema(t, db, name)
	assertAppRegistrationSchema(t, db, name)
	assertAppAuthorization(t, db, 2, 5, 8)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v1 LIKE '/v1/sys/app-members/%' AND v0<>'1'", 0)
	for _, filename := range []string{"20261019_app_members.sql", "20261019_app_registration_ip.sql"} {
		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(appSQL(t, "migrations/"+filename))))
		appCount(t, db, "SELECT COUNT(*) FROM schema_migrations WHERE filename=? AND checksum=?", 1, filename, checksum)
	}
}

func TestMySQLAppUpgradeRequiresExplicitAuthorizationAndIsRepeatable(t *testing.T) {
	before, _, found := strings.Cut(appSQL(t, "gozero-admin.sql"), appFreshMarker)
	if !found {
		t.Fatal("missing old installation boundary")
	}
	db, name := appMySQL(t, before)
	migration := appSQL(t, "migrations/20261019_app_members.sql")
	bootstrap := appSQL(t, "operator/20261019_app_member_admin_bootstrap.sql")
	appExec(t, db, migration)
	assertAppSchema(t, db, name)
	appExec(t, db, appSQL(t, "migrations/20261019_app_registration_ip.sql"))
	assertAppRegistrationSchema(t, db, name)
	assertAppAuthorization(t, db, 0, 0, 0)

	// A migration retry must preserve operator changes and must not duplicate seeds.
	appExec(t, db, "UPDATE app_membership_plans SET name='Customized VIP',enabled=0 WHERE code='vip' AND version=1")
	appExec(t, db, migration)
	assertAppAuthorization(t, db, 0, 0, 0)
	appCount(t, db, "SELECT COUNT(*) FROM app_membership_plans", 1)
	appCount(t, db, "SELECT COUNT(*) FROM app_membership_plans WHERE name='Customized VIP' AND enabled=0", 1)
	appCount(t, db, "SELECT COUNT(*) FROM sys_base_menus WHERE name IN('app-members-users','app-memberships')", 2)
	appCount(t, db, "SELECT COUNT(*) FROM sys_base_menu_btns WHERE permission_key LIKE 'app-members%'", 5)
	appCount(t, db, "SELECT COUNT(*) FROM sys_apis WHERE api_group='appmembers'", 8)

	// Future/custom routes under the same API group are not part of this explicit grant.
	appExec(t, db, "INSERT INTO sys_apis(created_at,updated_at,path,description,api_group,method) VALUES(NOW(3),NOW(3),'/v1/sys/app-members/custom-export','custom route','appmembers','POST')")
	appExec(t, db, bootstrap)
	appExec(t, db, bootstrap)
	assertAppAuthorization(t, db, 2, 5, 8)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v1='/v1/sys/app-members/custom-export'", 0)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v1 LIKE '/v1/sys/app-members/%' AND v0<>'1'", 0)
}

func TestMySQLAppRegistrationMigrationPreservesUsers(t *testing.T) {
	before, _, found := strings.Cut(appSQL(t, "gozero-admin.sql"), appFreshMarker)
	if !found {
		t.Fatal("missing old installation boundary")
	}
	db, name := appMySQL(t, before+appSQL(t, "migrations/20261019_app_members.sql"))
	appExec(t, db, "INSERT INTO app_users(id,nickname,status,session_version,review_reason,reviewed_by,created_at,updated_at) VALUES(1001,'Existing account','active',3,'Existing review',1,NOW(3),NOW(3))")
	registration := appSQL(t, "migrations/20261019_app_registration_ip.sql")
	appExec(t, db, registration)
	assertAppRegistrationSchema(t, db, name)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE id=1001 AND nickname='Existing account' AND status='active' AND session_version=3 AND review_reason='Existing review' AND reviewed_by=1 AND registration_ip='' AND registration_region=''", 1)

	// Retrying an additive migration must preserve subsequently collected IP data.
	appExec(t, db, "UPDATE app_users SET registration_ip='2001:db8::1',registration_region='Test region' WHERE id=1001")
	appExec(t, db, registration)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE id=1001 AND registration_ip='2001:db8::1' AND registration_region='Test region'", 1)
	appCount(t, db, "SELECT COUNT(*) FROM app_users", 1)
	assertAppRegistrationSchema(t, db, name)
	assertAppAuthorization(t, db, 0, 0, 0)

	// An interrupted deployment that added only the first column can safely resume.
	appExec(t, db, "ALTER TABLE app_users DROP COLUMN registration_region")
	appExec(t, db, registration)
	assertAppRegistrationSchema(t, db, name)
	appCount(t, db, "SELECT COUNT(*) FROM app_users WHERE id=1001 AND registration_ip='2001:db8::1' AND registration_region=''", 1)
}
