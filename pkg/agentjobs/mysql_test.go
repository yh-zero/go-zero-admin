package agentjobs

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"go-zero-admin/pkg/audit"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Only the random isolated database is opened; the DSN's original database is
// discarded. Requires CREATE/DROP DATABASE privileges, never migrates a user DB.
func mysqlFixture(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("GO_ZERO_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set GO_ZERO_MYSQL_TEST_DSN for isolated MySQL agent concurrency regression")
	}
	config, err := drivermysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal("invalid MySQL test DSN")
	}
	config.DBName = ""
	config.Timeout = 5 * time.Second
	config.ReadTimeout = 10 * time.Second
	config.WriteTimeout = 10 * time.Second
	admin, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatal("cannot connect to MySQL administration")
	}
	id, err := newID()
	if err != nil {
		_ = admin.Close()
		t.Fatal(err)
	}
	name := "gozero_agent_test_" + strings.ReplaceAll(id, "-", "")
	setup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := admin.ExecContext(setup, "CREATE DATABASE `"+name+"` CHARACTER SET utf8mb4"); err != nil {
		_ = admin.Close()
		t.Fatal("cannot create isolated agent database")
	}
	var connection *sql.DB
	t.Cleanup(func() {
		if connection != nil {
			_ = connection.Close()
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.ExecContext(ctx, "DROP DATABASE `"+name+"`"); err != nil {
			t.Error("cannot remove isolated agent database", err)
		}
		_ = admin.Close()
	})
	config.DBName = name
	config.ParseTime = true
	if config.Params == nil {
		config.Params = map[string]string{}
	}
	config.Params["transaction_isolation"] = "'REPEATABLE-READ'"
	db, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), DisableForeignKeyConstraintWhenMigrating: true})
	if err != nil {
		t.Fatal("cannot open isolated agent database")
	}
	connection, err = db.DB()
	if err != nil {
		t.Fatal(err)
	}
	connection.SetMaxOpenConns(10)
	if err := db.AutoMigrate(&fixtureOwner{}, &audit.Event{}); err != nil {
		t.Fatal(err)
	}
	// Use the immutable production migration's actual DDL rather than deriving
	// the three job tables from GORM, so these regressions catch model/schema drift.
	migration, err := os.ReadFile(filepath.Join("..", "..", "data", "db", "migrations", "20261003_zz_ai_agent.sql"))
	if err != nil {
		t.Fatal(err)
	}
	schemaDDL := strings.SplitN(strings.ReplaceAll(string(migration), "\r\n", "\n"), "SET NAMES utf8mb4", 2)[0]
	for _, statement := range strings.Split(schemaDDL, ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if err := db.Exec(statement).Error; err != nil {
			t.Fatal("cannot apply isolated agent schema", err)
		}
	}
	if err := db.Create(&fixtureOwner{ID: 1, Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestMySQLAppliedMigrationMatchesAgentModels(t *testing.T) {
	db := mysqlFixture(t)
	for _, model := range []any{&Conversation{}, &Message{}, &Run{}} {
		statement := &gorm.Statement{DB: db}
		if err := statement.Parse(model); err != nil {
			t.Fatal(err)
		}
		columns, err := db.Migrator().ColumnTypes(model)
		if err != nil {
			t.Fatal(err)
		}
		found := map[string]gorm.ColumnType{}
		for _, column := range columns {
			found[column.Name()] = column
		}
		for _, field := range statement.Schema.Fields {
			if field.DBName == "" {
				continue
			}
			column, exists := found[field.DBName]
			if !exists {
				t.Fatalf("%s.%s is missing", statement.Table, field.DBName)
			}
			if nullable, ok := column.Nullable(); ok && nullable && (field.NotNull || field.PrimaryKey) {
				t.Fatalf("%s.%s lost NOT NULL", statement.Table, field.DBName)
			}
			if field.Size > 0 && (field.DataType == "string") {
				if length, ok := column.Length(); ok && length != int64(field.Size) {
					t.Fatalf("%s.%s length=%d model=%d", statement.Table, field.DBName, length, field.Size)
				}
			}
		}
		if len(found) != len(statement.Schema.Fields)-ignoredFields(statement) {
			t.Fatalf("%s has unexpected columns", statement.Table)
		}
	}
	for _, index := range []struct {
		model any
		name  string
	}{{&Run{}, "uk_agent_run_request"}, {&Run{}, "uk_agent_run_sequence"}, {&Message{}, "uk_agent_message_run_role"}, {&Run{}, "idx_agent_run_claim"}, {&Run{}, "idx_agent_run_lease"}, {&Run{}, "idx_agent_run_owner_status"}, {&Run{}, "idx_agent_run_conversation_status"}, {&Message{}, "idx_agent_message_history"}, {&Conversation{}, "idx_agent_conversation_owner"}} {
		if !db.Migrator().HasIndex(index.model, index.name) {
			t.Fatal("production index missing", index.name)
		}
	}
	var binaryIDs int64
	if err := db.Raw("SELECT COUNT(*) FROM information_schema.columns WHERE table_schema=DATABASE() AND table_name IN ('sys_ai_conversations','sys_ai_messages','sys_ai_runs') AND column_name IN ('id','request_id','request_hash','conversation_id','run_id','session_id','lease_owner') AND collation_name='utf8mb4_bin'").Scan(&binaryIDs).Error; err != nil {
		t.Fatal(err)
	}
	if binaryIDs != 10 {
		t.Fatalf("binary identifier columns=%d, want 10", binaryIDs)
	}
}

func ignoredFields(statement *gorm.Statement) int {
	count := 0
	for _, field := range statement.Schema.Fields {
		if field.DBName == "" {
			count++
		}
	}
	return count
}

func TestMySQLConcurrentSubmitDedupLimitAndClaim(t *testing.T) {
	db := mysqlFixture(t)
	first := manager(t, db, nil)
	second := manager(t, db, nil)
	input := request(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var group sync.WaitGroup
	replies := make(chan Run, 8)
	failures := make(chan error, 8)
	for index := 0; index < 8; index++ {
		group.Add(1)
		go func(index int) {
			defer group.Done()
			scheduler := first
			if index%2 != 0 {
				scheduler = second
			}
			run, err := scheduler.Submit(ctx, input)
			if err != nil {
				failures <- err
			} else {
				replies <- run
			}
		}(index)
	}
	group.Wait()
	close(replies)
	close(failures)
	for err := range failures {
		t.Fatal("concurrent dedup failed", err)
	}
	var original Run
	for run := range replies {
		if original.ID == "" {
			original = run
		}
		if run.ID != original.ID || run.ConversationID != original.ConversationID {
			t.Fatal("dedup created different runs")
		}
	}
	var success, busy atomic.Int64
	unexpected := make(chan error, 8)
	for index := 0; index < 8; index++ {
		next := request(t)
		group.Add(1)
		go func() {
			defer group.Done()
			_, err := second.Submit(ctx, next)
			if err == nil {
				success.Add(1)
			} else if errors.Is(err, ErrBusy) {
				busy.Add(1)
			} else {
				unexpected <- err
			}
		}()
	}
	group.Wait()
	close(unexpected)
	for err := range unexpected {
		t.Fatal("concurrent limit failed", err)
	}
	if success.Load() != 1 || busy.Load() != 7 {
		t.Fatal("user limit not serialized", success.Load(), busy.Load())
	}
	var runCount, messageCount int64
	db.Model(&Run{}).Count(&runCount)
	db.Model(&Message{}).Count(&messageCount)
	if runCount != 2 || messageCount != 2 {
		t.Fatal("dedup left extra data", runCount, messageCount)
	}
	claimed := make(chan string, 4)
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	var calls sync.Map
	execute := func(ctx context.Context, job Job) (Result, error) {
		counter, _ := calls.LoadOrStore(job.Run.ID, &atomic.Int64{})
		counter.(*atomic.Int64).Add(1)
		claimed <- job.Run.ID
		select {
		case <-release:
			return Result{Text: "done"}, nil
		case <-ctx.Done():
			return Result{}, ctx.Err()
		}
	}
	first.execute = execute
	second.execute = execute
	first.cfg.RunTimeout = 3 * time.Second
	second.cfg.RunTimeout = 3 * time.Second
	if err := first.Start(); err != nil {
		t.Fatal(err)
	}
	if err := second.Start(); err != nil {
		t.Fatal(err)
	}
	var ids []string
	for index := 0; index < 2; index++ {
		select {
		case id := <-claimed:
			ids = append(ids, id)
		case <-ctx.Done():
			t.Fatal("two workers did not claim distinct rows")
		}
	}
	if ids[0] == ids[1] {
		t.Fatal("two workers executed same run")
	}
	if _, err := first.Cancel(ctx, 1, original.ID); err != nil {
		t.Fatal(err)
	}
	releaseOnce.Do(func() { close(release) })
	waitRun(t, first, original.ID, StatusCanceled)
	otherID := ids[0]
	if otherID == original.ID {
		otherID = ids[1]
	}
	waitRun(t, first, otherID, StatusSucceeded)
	calls.Range(func(_ any, value any) bool {
		if value.(*atomic.Int64).Load() != 1 {
			t.Error("provider invoked repeatedly")
		}
		return true
	})
	var assistants int64
	db.Model(&Message{}).Where("role=?", "assistant").Count(&assistants)
	if assistants != 1 {
		t.Fatal("late canceled result wrote assistant message", assistants)
	}
}

func TestMySQLDifferentOwnersCanSubmitIntoEmptyIndexes(t *testing.T) {
	db := mysqlFixture(t)
	if err := db.Create(&fixtureOwner{ID: 2, Enable: 1}).Error; err != nil {
		t.Fatal(err)
	}
	first := manager(t, db, nil)
	second := manager(t, db, nil)
	ready, release := make(chan struct{}), make(chan struct{})
	var arrivals atomic.Int64
	var once sync.Once
	if err := db.Callback().Create().Before("gorm:create").Register("test:both-owners-after-empty-range-reads", func(tx *gorm.DB) {
		if tx.Statement.Table != "sys_ai_conversations" {
			return
		}
		if arrivals.Add(1) == 2 {
			close(ready)
		}
		select {
		case <-release:
		case <-tx.Statement.Context.Done():
			tx.AddError(tx.Statement.Context.Err())
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		once.Do(func() { close(release) })
		_ = db.Callback().Create().Remove("test:both-owners-after-empty-range-reads")
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	inputs := []SubmitInput{request(t), request(t)}
	inputs[1].OwnerID = 2
	results := make(chan error, 2)
	go func() { _, err := first.Submit(ctx, inputs[0]); results <- err }()
	go func() { _, err := second.Submit(ctx, inputs[1]); results <- err }()
	select {
	case <-ready:
	case <-ctx.Done():
		t.Fatal("owners did not reach empty-index barrier")
	}
	once.Do(func() { close(release) })
	for index := 0; index < 2; index++ {
		if err := <-results; err != nil {
			t.Fatal("unrelated owners deadlocked on empty request/active ranges", err)
		}
	}
}
