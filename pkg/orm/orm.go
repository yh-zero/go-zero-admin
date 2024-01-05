package orm

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

type Config struct {
	DSN          string
	MaxOpenConns int
	MaxIdleCnns  int
	MaxLifetime  int
}

type DB struct {
	*gorm.DB
}

type ormLog struct {
	LogLevel logger.LogLevel
}

func (l *ormLog) LogMode(level logger.LogLevel) logger.Interface {
	l.LogLevel = level
	return l

}

func (l *ormLog) Info(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel < logger.Info {
		return
	}
	logx.WithContext(ctx).Infof(s, i...)
}

func (l *ormLog) Warn(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel < logger.Warn {
		return
	}
	logx.WithContext(ctx).Infof(s, i...)
}

func (l *ormLog) Error(ctx context.Context, s string, i ...interface{}) {
	if l.LogLevel < logger.Error {
		return
	}
	logx.WithContext(ctx).Infof(s, i...)
}

func (l *ormLog) Trace(ctx context.Context, begin time.Time, fc func() (sql string, rowsAffected int64), err error) {
	elapsed := time.Since(begin)
	sql, rows := fc()
	logx.WithContext(ctx).WithDuration(elapsed).Infof("[%.3fms] [rows: %v] %s", float64(elapsed.Nanoseconds())/1e6, rows, sql)
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

	db, err := gorm.Open(mysql.Open(conf.DSN), &gorm.Config{Logger: &ormLog{}})
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
