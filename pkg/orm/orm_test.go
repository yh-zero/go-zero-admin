package orm

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	drivermysql "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func TestTraceHonorsLevelsAndRedactsValues(t *testing.T) {
	var levels []logger.LogLevel
	var messages []string
	l := &ormLog{LogLevel: logger.Warn, SlowThreshold: 10 * time.Millisecond, emit: func(_ context.Context, level logger.LogLevel, _ time.Duration, message string) {
		levels = append(levels, level)
		messages = append(messages, message)
	}}
	called := false
	fc := func() (string, int64) {
		called = true
		return `SELECT * FROM users WHERE email='private@example.com' AND phone="12345678901" AND id=123 /* password secret */`, 1
	}
	l.Trace(context.Background(), time.Now(), fc, nil)
	if called {
		t.Fatal("fast query should not be formatted below Info level")
	}
	l.Trace(context.Background(), time.Now().Add(-20*time.Millisecond), fc, nil)
	l.Trace(context.Background(), time.Now(), fc, &drivermysql.MySQLError{Number: 1062, Message: "Duplicate entry 'password secret'"})
	if len(levels) != 2 || levels[0] != logger.Warn || levels[1] != logger.Error {
		t.Fatalf("wrong levels: %v", levels)
	}
	for _, message := range messages {
		for _, secret := range []string{"private@example.com", "12345678901", "id=123", "password secret"} {
			if strings.Contains(message, secret) {
				t.Fatalf("log exposed %q: %s", secret, message)
			}
		}
	}
	called = false
	l.LogMode(logger.Silent).Trace(context.Background(), time.Now(), fc, context.Canceled)
	if called {
		t.Fatal("Silent logger called SQL formatter")
	}
	if l.LogLevel != logger.Warn {
		t.Fatal("LogMode mutated the shared logger")
	}
}

func TestGORMLoggerPreservesPlaceholders(t *testing.T) {
	var message string
	l := &ormLog{LogLevel: logger.Info, emit: func(_ context.Context, _ logger.LogLevel, _ time.Duration, output string) { message = output }}
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: l})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	var value string
	if err := db.Raw("SELECT ?", "private-password").Scan(&value).Error; err != nil {
		t.Fatal(err)
	}
	if value != "private-password" {
		t.Fatal("redaction changed query execution")
	}
	if strings.Contains(message, "private-password") || !strings.Contains(message, "SELECT ?") {
		t.Fatalf("unexpected SQL log: %s", message)
	}
}

func TestPluginStartsBeforeDatabaseOperationWithSubsecondPrecision(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	if err := db.Use(NewCustomePlugin()); err != nil {
		t.Fatal(err)
	}
	started := false
	if err := db.Callback().Raw().Before("gorm:raw").Register("test:probe", func(tx *gorm.DB) {
		value, exists := tx.InstanceGet("gorm:operation_trace:raw")
		if !exists {
			t.Error("trace did not start before the SQL operation")
			return
		}
		state, ok := value.(operationTrace)
		if !ok || state.start.IsZero() || state.start.Nanosecond() == 0 {
			t.Errorf("imprecise start time: %#v", value)
			return
		}
		started = true
	}); err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("CREATE TABLE plugin_probe (id INTEGER)").Error; err != nil {
		t.Fatal(err)
	}
	if !started {
		t.Fatal("operation callback did not run")
	}
}

func TestRedactEscapedRawLiterals(t *testing.T) {
	input := "SELECT `field2026`, column1, 'private''value', 'escaped\\'secret', 1.23e-4, 0xABC -- hidden\nFROM `users` WHERE value=? # hidden"
	result := redactSQL(input)
	for _, secret := range []string{"private", "escaped", "secret", "1.23", "0xABC", "hidden"} {
		if strings.Contains(result, secret) {
			t.Fatalf("literal leaked: %s", result)
		}
	}
	if !strings.Contains(result, "`field2026`") || !strings.Contains(result, "column1") {
		t.Fatalf("identifiers were damaged: %s", result)
	}
}
