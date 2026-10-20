package db_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
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
	assertAppAuthorization(t, db, 2, 5, 8)
	appCount(t, db, "SELECT COUNT(*) FROM casbin_rule WHERE ptype='p' AND v1 LIKE '/v1/sys/app-members/%' AND v0<>'1'", 0)
	for _, filename := range []string{"20261019_app_members.sql", "20261019_app_registration_ip.sql"} {
		checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(appSQL(t, "migrations/"+filename))))
		appCount(t, db, "SELECT COUNT(*) FROM schema_migrations WHERE filename=? AND checksum=?", 1, filename, checksum)
	}
}
