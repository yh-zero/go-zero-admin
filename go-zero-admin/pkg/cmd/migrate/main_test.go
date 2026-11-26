package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
)

func TestPositionalActionBeforeFlags(t *testing.T) {
	action, args, err := splitAction([]string{"down", "-yes"})
	if err != nil || action != "down" || len(args) != 1 || args[0] != "-yes" {
		t.Fatalf("位置参数解析错误: action=%s args=%v err=%v", action, args, err)
	}
	action, args, err = splitAction([]string{"-f", "custom.yaml", "force", "-version", "2", "-yes"})
	if err != nil || action != "force" || len(args) != 5 {
		t.Fatalf("混合参数解析错误: action=%s args=%v err=%v", action, args, err)
	}
}

func TestActionFlagValueIsNotPositional(t *testing.T) {
	for _, args := range [][]string{{"-action", "down", "-yes"}, {"-action", "force", "-version", "2", "-yes"}, {"-f", "up", "status"}} {
		action, remaining, err := splitAction(args)
		if err != nil {
			t.Fatal(err)
		}
		if len(args) == 3 && args[0] == "-f" && action != "status" {
			t.Fatalf("位置动作解析错误: action=%s", action)
		}
		if args[0] == "-action" && action != "" {
			t.Fatalf("-action 的值被误识别为位置参数: action=%s", action)
		}
		if len(remaining) != len(args)-1 && action != "" {
			t.Fatalf("位置参数移除错误: %v", remaining)
		}
	}
}

func TestDataSourceEnvOverride(t *testing.T) {
	configFile := filepath.Join(t.TempDir(), "migrate.yaml")
	if err := os.WriteFile(configFile, []byte("Mysql:\n  DataSource: config-dsn\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MYSQL_DATASOURCE", "env-dsn")
	dsn, err := loadDataSource(configFile)
	if err != nil || dsn != "env-dsn" {
		t.Fatalf("环境变量未覆盖配置: dsn=%s err=%v", dsn, err)
	}
	t.Setenv("MYSQL_DATASOURCE", "")
	dsn, err = loadDataSource(configFile)
	if err != nil || dsn != "config-dsn" {
		t.Fatalf("配置文件默认值错误: dsn=%s err=%v", dsn, err)
	}
}

func TestMigrationURLs(t *testing.T) {
	if got := mysqlURL("root:123456@tcp(mysql:3306)/gozero_admin"); got != "mysql://root:123456@tcp(mysql:3306)/gozero_admin" {
		t.Fatalf("mysql URL mismatch: %s", got)
	}
	if got := fileURL("data/db/migrations"); got != "file://data/db/migrations" {
		t.Fatalf("file URL mismatch: %s", got)
	}
}

func TestMigrationBaselineAndFutureUpgrade(t *testing.T) {
	t.Setenv("MYSQL_DATASOURCE", "")
	rootDSN := os.Getenv("GOZERO_MIGRATE_TEST_DSN")
	if rootDSN == "" {
		t.Skip("需要隔离的 GOZERO_MIGRATE_TEST_DSN")
	}
	rootConfig, err := mysqlDriver.ParseDSN(rootDSN)
	if err != nil {
		t.Fatal(err)
	}
	dbName := fmt.Sprintf("gozero_migrate_test_%d", time.Now().UnixNano())
	rootConfig.DBName = ""
	root, err := sql.Open("mysql", rootConfig.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer root.Close()
	if _, err := root.Exec("CREATE DATABASE `" + dbName + "` CHARACTER SET utf8mb4 COLLATE=utf8mb4_0900_ai_ci"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = root.Exec("DROP DATABASE IF EXISTS `" + dbName + "`") })

	testConfig := *rootConfig
	testConfig.DBName = dbName
	testConfig.MultiStatements = true
	db, err := sql.Open("mysql", testConfig.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	initSQL, err := os.ReadFile(filepath.Join("..", "..", "..", "data", "db", "init", "mysql-all.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(withoutDatabaseSelection(string(initSQL))); err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(t.TempDir(), "migrate.yaml")
	if err := os.WriteFile(configFile, []byte("Mysql:\n  DataSource: "+strconv.Quote(testConfig.FormatDSN())+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	baseDir := filepath.Join("..", "..", "..", "data", "db", "migrations")
	opts := options{configFile: configFile, dir: baseDir, steps: 1}
	ctx := context.Background()
	opts.action = "up"
	if err := run(ctx, opts); err != nil {
		t.Fatalf("新库基线迁移失败: %v", err)
	}
	assertVersion(t, db, 2)

	futureDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(futureDir, "000002_baseline.down.sql"), []byte("-- baseline\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(futureDir, "000003_future.up.sql"), []byte("CREATE TABLE migration_probe (id bigint PRIMARY KEY) ENGINE=InnoDB;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(futureDir, "000003_future.down.sql"), []byte("DROP TABLE IF EXISTS migration_probe;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts.dir = futureDir
	if err := run(ctx, opts); err != nil {
		t.Fatalf("未来版本迁移失败: %v", err)
	}
	assertVersion(t, db, 3)
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name='migration_probe'", dbName).Scan(&count); err != nil || count != 1 {
		t.Fatalf("未来迁移表未创建: %d %v", count, err)
	}
	opts.action = "down"
	if err := run(ctx, opts); err == nil {
		t.Fatal("未确认的 down 必须被拒绝")
	}
	opts.yes = true
	if err := run(ctx, opts); err != nil {
		t.Fatalf("未来版本回退失败: %v", err)
	}
	assertVersion(t, db, 2)
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name='migration_probe'", dbName).Scan(&count); err != nil || count != 0 {
		t.Fatalf("未来迁移表未回退: %d %v", count, err)
	}
	opts.action = "up"
	opts.yes = false
	if err := run(ctx, opts); err != nil {
		t.Fatalf("未来版本再次升级失败: %v", err)
	}
	assertVersion(t, db, 3)
	opts.action = "force"
	opts.version = 2
	if err := run(ctx, opts); err == nil {
		t.Fatal("未确认的 force 必须被拒绝")
	}
	opts.yes = true
	if err := run(ctx, opts); err != nil {
		t.Fatalf("force 修复版本失败: %v", err)
	}
	assertVersion(t, db, 2)
	opts.action = "status"
	if err := run(ctx, opts); err != nil {
		t.Fatalf("读取迁移状态失败: %v", err)
	}

	if err := os.WriteFile(filepath.Join(futureDir, "000003_future.up.sql"), []byte("THIS IS NOT VALID SQL;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts.action = "up"
	opts.yes = false
	if err := run(ctx, opts); err == nil {
		t.Fatal("故意损坏的迁移必须失败")
	}
	assertDirtyVersion(t, db, 3)
	opts.action = "force"
	opts.version = 2
	opts.yes = true
	if err := run(ctx, opts); err != nil {
		t.Fatalf("dirty 状态修复失败: %v", err)
	}
	assertVersion(t, db, 2)
}

func withoutDatabaseSelection(sqlText string) string {
	lines := strings.Split(sqlText, "\n")
	kept := lines[:0]
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "CREATE DATABASE") || strings.HasPrefix(trimmed, "USE ") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}

func assertVersion(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	var version int
	var dirty bool
	if err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil || version != expected || dirty {
		t.Fatalf("版本状态错误: version=%d dirty=%t err=%v", version, dirty, err)
	}
}

func assertDirtyVersion(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	var version int
	var dirty bool
	if err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil || version != expected || !dirty {
		t.Fatalf("dirty 状态错误: version=%d dirty=%t err=%v", version, dirty, err)
	}
}
