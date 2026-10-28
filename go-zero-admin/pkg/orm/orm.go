package orm

import (
	"context"
	"errors"
	"fmt"
	"time"

	drivermysql "github.com/go-sql-driver/mysql"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Config struct {
	DSN           string
	MaxOpenConns  int
	MaxIdleCnns   int
	MaxLifetime   int
	LogLevel      logger.LogLevel
	SlowThreshold time.Duration
}

type DB struct {
	*gorm.DB
}

type ormLog struct {
	LogLevel      logger.LogLevel
	SlowThreshold time.Duration
	// emit is overridden only by tests; runtime output uses the contextual logger.
	emit func(context.Context, logger.LogLevel, time.Duration, string)
}

func (l *ormLog) LogMode(level logger.LogLevel) logger.Interface {
	copy := *l
	copy.LogLevel = level
	return &copy
}

func (l *ormLog) Info(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel < logger.Info {
		return
	}
	l.write(ctx, logger.Info, 0, fmt.Sprintf(s, i...))
}

func (l *ormLog) Warn(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel < logger.Warn {
		return
	}
	l.write(ctx, logger.Warn, 0, fmt.Sprintf(s, i...))
}

func (l *ormLog) Error(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel < logger.Error {
		return
	}
	l.write(ctx, logger.Error, 0, fmt.Sprintf(s, i...))
}

func (l *ormLog) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	if l.LogLevel == logger.Silent {
		return
	}
	elapsed := time.Since(begin)
	level := logger.Info
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		level = logger.Error
	} else if l.SlowThreshold > 0 && elapsed > l.SlowThreshold {
		level = logger.Warn
	}
	if l.LogLevel < level {
		return
	}
	sql, rows := fc()
	message := fmt.Sprintf("[%.3fms] [rows: %v] %s", float64(elapsed)/float64(time.Millisecond), rows, redactSQL(sql))
	if level == logger.Error {
		// Database errors (especially duplicate-key errors) can contain submitted values.
		var mysqlErr *drivermysql.MySQLError
		switch {
		case errors.As(err, &mysqlErr):
			message += fmt.Sprintf(" [mysql error: %d]", mysqlErr.Number)
		case errors.Is(err, context.Canceled):
			message += " [context canceled]"
		case errors.Is(err, context.DeadlineExceeded):
			message += " [deadline exceeded]"
		default:
			message += fmt.Sprintf(" [error type: %T]", err)
		}
	}
	l.write(ctx, level, elapsed, message)
}

// ParamsFilter makes GORM retain placeholders instead of interpolating arguments.
func (l *ormLog) ParamsFilter(_ context.Context, sql string, _ ...interface{}) (string, []interface{}) {
	return sql, nil
}

func (l *ormLog) write(ctx context.Context, level logger.LogLevel, duration time.Duration, message string) {
	if l.emit != nil {
		l.emit(ctx, level, duration, message)
		return
	}
	output := logx.WithContext(ctx).WithDuration(duration)
	switch level {
	case logger.Error:
		output.Error(message)
	case logger.Warn:
		output.Slow(message)
	default:
		output.Info(message)
	}
}

func NewMysql(conf *Config) (*DB, error) {
	if conf.MaxIdleCnns == 0 {
		conf.MaxIdleCnns = 10
	}
	if conf.MaxOpenConns == 0 {
		conf.MaxOpenConns = 100
	}
	if conf.MaxLifetime == 0 {
		conf.MaxLifetime = 3600
	}

	if conf.LogLevel == 0 {
		conf.LogLevel = logger.Warn
	}
	if conf.SlowThreshold == 0 {
		conf.SlowThreshold = 200 * time.Millisecond
	}
	db, err := gorm.Open(mysql.Open(conf.DSN), &gorm.Config{Logger: &ormLog{LogLevel: conf.LogLevel, SlowThreshold: conf.SlowThreshold}})
	if err != nil {
		return nil, err
	}
	sdb, err := db.DB()
	if err != nil {
		return nil, err
	}
	sdb.SetMaxIdleConns(conf.MaxIdleCnns)
	sdb.SetMaxOpenConns(conf.MaxOpenConns)
	sdb.SetConnMaxLifetime(time.Second * time.Duration(conf.MaxLifetime))

	err = db.Use(NewCustomePlugin())

	if err != nil {
		return nil, err
	}

	return &DB{DB: db}, nil

}

func MustNewMysql(conf *Config) *DB {
	db, err := NewMysql(conf)
	if err != nil {
		panic(err)
	}
	return db
}
