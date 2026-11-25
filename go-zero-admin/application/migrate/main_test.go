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

func TestMigrationURLs(t *testing.T) {
	if got := mysqlURL("root:123456@tcp(mysql:3306)/gozero_admin"); got != "mysql://root:123456@tcp(mysql:3306)/gozero_admin" {
		t.Fatalf("mysql URL mismatch: %s", got)
	}
	if got := fileURL("data/db/migrations"); got != "file://data/db/migrations" {
		t.Fatalf("file URL mismatch: %s", got)
	}
}

func TestMigrationAgainstMySQL(t *testing.T) {
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
	initSQL, err := os.ReadFile(filepath.Join("..", "..", "data", "db", "init", "mysql-all.sql"))
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
	opts := options{configFile: configFile, dir: filepath.Join("..", "..", "data", "db", "migrations"), steps: 1}
	ctx := context.Background()
	opts.action = "up"
	if err := run(ctx, opts); err != nil {
		t.Fatalf("首次执行迁移失败: %v", err)
	}
	assertVersion(t, db, 2)
	var tables int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name IN ('sys_permission','sys_policy_state')", dbName).Scan(&tables); err != nil || tables != 2 {
		t.Fatalf("权限表未创建: tables=%d err=%v", tables, err)
	}
	if err := run(ctx, opts); err != nil {
		t.Fatalf("重复执行迁移失败: %v", err)
	}
	assertVersion(t, db, 2)

	opts.action = "down"
	if err := run(ctx, opts); err == nil {
		t.Fatal("未确认的 down 必须被拒绝")
	}
	opts.yes = true
	if err := run(ctx, opts); err != nil {
		t.Fatalf("回退权限迁移失败: %v", err)
	}
	assertVersion(t, db, 1)
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=? AND table_name='sys_permission'", dbName).Scan(&tables); err != nil || tables != 0 {
		t.Fatalf("权限表未回退: tables=%d err=%v", tables, err)
	}
	if err := run(ctx, opts); err != nil {
		t.Fatalf("回退文件迁移失败: %v", err)
	}
	assertVersion(t, db, 0)
	var columns int
	if err := db.QueryRow("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=? AND table_name='sys_file' AND column_name='sha256'", dbName).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("sha256 未回退: columns=%d err=%v", columns, err)
	}
	opts.action = "up"
	opts.yes = false
	if err := run(ctx, opts); err != nil {
		t.Fatalf("旧库重新升级失败: %v", err)
	}
	assertVersion(t, db, 2)
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

	brokenDir := filepath.Join(t.TempDir(), "broken-migrations")
	if err := copyMigrationFiles(opts.dir, brokenDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, "000003_broken.up.sql"), []byte("THIS IS NOT VALID SQL;\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	opts.dir = brokenDir
	opts.action = "up"
	opts.yes = false
	if err := run(ctx, opts); err == nil {
		t.Fatal("故意损坏的迁移必须失败")
	}
	assertDirtyVersion(t, db, 3)
	opts.action = "status"
	if err := run(ctx, opts); err != nil {
		t.Fatalf("dirty 状态读取失败: %v", err)
	}
	opts.action = "force"
	opts.version = 2
	opts.yes = true
	if err := run(ctx, opts); err != nil {
		t.Fatalf("dirty 状态修复失败: %v", err)
	}
	assertVersion(t, db, 2)
}

func copyMigrationFiles(sourceDir, targetDir string) error {
	if err := os.MkdirAll(targetDir, 0o700); err != nil {
		return err
	}
	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(sourceDir, entry.Name()))
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(targetDir, entry.Name()), data, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func assertDirtyVersion(t *testing.T, db *sql.DB, expected int) {
	t.Helper()
	var version int
	var dirty bool
	if err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil || version != expected || !dirty {
		t.Fatalf("dirty 状态错误: version=%d dirty=%t err=%v", version, dirty, err)
	}
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
	err := db.QueryRow("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty)
	if expected == 0 {
		if err == nil {
			t.Fatalf("期望无迁移版本，实际 version=%d dirty=%t", version, dirty)
		}
		return
	}
	if err != nil || version != expected || dirty {
		t.Fatalf("版本状态错误: version=%d dirty=%t err=%v", version, dirty, err)
	}
}
