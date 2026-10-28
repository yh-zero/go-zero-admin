// Package testmysql provides isolated, opt-in MySQL fixtures. It never opens the configured business database.
package testmysql

import (
	"context"
	"database/sql"
	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/gofrs/uuid/v5"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
	"strings"
	"testing"
	"time"
)

func Open(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("GO_ZERO_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set GO_ZERO_MYSQL_TEST_DSN to run the isolated MySQL concurrency regression")
	}
	config, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid GO_ZERO_MYSQL_TEST_DSN")
	}
	config.DBName = ""
	config.Timeout, config.ReadTimeout, config.WriteTimeout = 5*time.Second, 15*time.Second, 15*time.Second
	admin, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal("cannot open MySQL test administration connection")
	}
	admin.SetMaxOpenConns(1)
	id, err := uuid.NewV4()
	if err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	const prefix = "gozero_test_rr_"
	name := prefix + strings.ReplaceAll(id.String(), "-", "")
	if !strings.HasPrefix(name, prefix) || len(name) != len(prefix)+32 || strings.ContainsAny(name, "`\\/ \x00") {
		_ = admin.Close()
		t.Fatal("invalid isolated test database identifier")
	}
	setupCtx, setupCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer setupCancel()
	if _, err = admin.ExecContext(setupCtx, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4"); err != nil {
		_ = admin.Close()
		t.Fatal("cannot create isolated MySQL test database: ", err)
	}
	var database *sql.DB
	t.Cleanup(func() {
		if database != nil {
			_ = database.Close()
		}
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(cleanupCtx, "DROP DATABASE `"+name+"`"); err != nil {
			t.Errorf("cannot remove isolated test database %s: %v", name, err)
		}
		_ = admin.Close()
	})
	config.DBName, config.ParseTime = name, true
	if config.Params == nil {
		config.Params = map[string]string{}
	}
	// Explicitly exercise the default InnoDB isolation, regardless of host settings.
	config.Params["transaction_isolation"] = "'REPEATABLE-READ'"
	db, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger:                                   logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		t.Fatal("cannot open isolated MySQL test database: ", err)
	}
	database, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(4)

	return db
}
