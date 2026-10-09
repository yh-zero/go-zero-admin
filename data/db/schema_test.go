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

	drivermysql "github.com/go-sql-driver/mysql"
	"go-zero-admin/pkg/testmysql"
	"gorm.io/driver/mysql"
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

// Exercise the actual fresh-install SQL, including its seeds and ALTERs, in an
// isolated schema. This catches drift that AutoMigrate-only fixtures cannot.
func TestMySQLFreshPermissionSchema(t *testing.T) {
	db := testmysql.Open(t)
	dialector, ok := db.Dialector.(*mysql.Dialector)
	if !ok {
		t.Fatal("expected MySQL fixture")
	}
	config, err := drivermysql.ParseDSN(dialector.DSN)
	if err != nil {
		t.Fatal("invalid isolated fixture DSN")
	}
	if !strings.HasPrefix(config.DBName, "gozero_test_rr_") {
		t.Fatal("fresh schema requires isolated test database")
	}
	config.MultiStatements = true
	client, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	source, err := os.ReadFile("gozero-admin.sql")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = client.Exec(string(source)); err != nil {
		t.Fatalf("fresh schema import failed: %v", err)
	}
	var count int
	for _, table := range []string{"sys_permission_versions", "sys_permission_changes"} {
		if err = client.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?", config.DBName, table).Scan(&count); err != nil || count != 1 {
			t.Fatalf("missing %s: %d %v", table, count, err)
		}
	}
	if err = client.QueryRow("SELECT COUNT(*) FROM sys_base_menu_btns WHERE permission_key IS NULL OR permission_key = ''").Scan(&count); err != nil || count != 0 {
		t.Fatalf("stable button keys missing: %d %v", count, err)
	}
	if err = client.QueryRow("SELECT COUNT(*) FROM casbin_rule WHERE ptype = 'p' AND v0 = '1' AND v1 = '/v1/sys/permissions/edit' AND v2 = 'GET'").Scan(&count); err != nil || count != 1 {
		t.Fatalf("administrator editor missing: %d %v", count, err)
	}
	migration, err := os.ReadFile("migrations/20261008_permission_revision_history.sql")
	if err != nil {
		t.Fatal(err)
	}
	checksum := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.ReplaceAll(string(migration), "\r\n", "\n"))))
	var recorded string
	if err = client.QueryRow("SELECT checksum FROM schema_migrations WHERE filename = '20261008_permission_revision_history.sql'").Scan(&recorded); err != nil || recorded != checksum {
		t.Fatalf("fresh schema migration record mismatch: %v", err)
	}
}
