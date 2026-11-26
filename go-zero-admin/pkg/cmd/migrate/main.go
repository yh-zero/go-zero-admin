package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	"github.com/golang-migrate/migrate/v4/source"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/zeromicro/go-zero/core/conf"
)

type options struct {
	configFile string
	dir        string
	action     string
	version    int
	steps      int
	yes        bool
}

type migrationConfig struct {
	Mysql struct {
		DataSource string `json:"DataSource"`
	} `json:"Mysql"`
}

type migrateLogger struct{}

func (migrateLogger) Printf(format string, args ...interface{}) { log.Printf(format, args...) }
func (migrateLogger) Verbose() bool                             { return false }

func main() {
	opts := options{}
	flag.StringVar(&opts.configFile, "f", "pkg/cmd/migrate/etc/migrate.yaml", "迁移配置文件")
	flag.StringVar(&opts.dir, "dir", "data/db/migrations", "迁移文件目录")
	flag.StringVar(&opts.action, "action", "up", "执行动作：up、status、down、force")
	flag.IntVar(&opts.version, "version", 0, "force 动作的目标版本")
	flag.IntVar(&opts.steps, "steps", 1, "down 动作的回退步数")
	flag.BoolVar(&opts.yes, "yes", false, "确认执行 down 或 force")
	action, args, err := splitAction(os.Args[1:])
	if err != nil {
		log.Fatal(err)
	}
	if err := flag.CommandLine.Parse(args); err != nil {
		log.Fatal(err)
	}
	if flag.NArg() != 0 {
		log.Fatal("存在无法识别的位置参数")
	}
	if action != "" {
		opts.action = action
	}
	if err := run(context.Background(), opts); err != nil {
		log.Fatal(err)
	}
}

func splitAction(args []string) (string, []string, error) {
	action := ""
	skipValue := false
	for i, arg := range args {
		if skipValue {
			skipValue = false
			continue
		}
		switch arg {
		case "-f", "-dir", "-action", "-version", "-steps":
			skipValue = true
			continue
		case "up", "status", "down", "force":
			if action != "" {
				return "", nil, fmt.Errorf("只允许一个位置参数")
			}
			action = arg
			args = append(args[:i:i], args[i+1:]...)
			return action, args, nil
		}
	}
	return "", args, nil
}

func loadDataSource(configFile string) (string, error) {
	var cfg migrationConfig
	if err := conf.Load(configFile, &cfg, conf.UseEnv()); err != nil {
		return "", fmt.Errorf("读取迁移配置: %w", err)
	}
	if dsn := os.Getenv("MYSQL_DATASOURCE"); dsn != "" {
		return dsn, nil
	}
	if cfg.Mysql.DataSource == "" {
		return "", fmt.Errorf("Mysql.DataSource 不能为空，或通过 MYSQL_DATASOURCE 指定")
	}
	return cfg.Mysql.DataSource, nil
}

func run(ctx context.Context, opts options) error {
	if opts.configFile == "" {
		return fmt.Errorf("迁移配置文件不能为空")
	}
	if opts.dir == "" {
		return fmt.Errorf("迁移文件目录不能为空")
	}
	if opts.steps <= 0 {
		return fmt.Errorf("steps 必须大于 0")
	}
	dsn, err := loadDataSource(opts.configFile)
	if err != nil {
		return err
	}
	sourceDriver, err := source.Open(fileURL(opts.dir))
	if err != nil {
		return fmt.Errorf("打开迁移文件: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("file", sourceDriver, mysqlURL(dsn))
	if err != nil {
		_ = sourceDriver.Close()
		return fmt.Errorf("初始化迁移器: %w", err)
	}
	m.Log = migrateLogger{}
	defer func() {
		sourceErr, databaseErr := m.Close()
		if sourceErr != nil {
			log.Printf("关闭迁移文件失败: %v", sourceErr)
		}
		if databaseErr != nil {
			log.Printf("关闭数据库连接失败: %v", databaseErr)
		}
	}()
	if err := ctx.Err(); err != nil {
		return err
	}
	switch opts.action {
	case "up":
		err = m.Up()
		if errors.Is(err, migrate.ErrNoChange) {
			log.Printf("数据库已经是最新迁移版本")
			err = nil
		}
	case "status":
		return printStatus(m, sourceDriver)
	case "down":
		if !opts.yes {
			return fmt.Errorf("down 会撤销数据库迁移，必须显式指定 -yes")
		}
		err = m.Steps(-opts.steps)
	case "force":
		if !opts.yes {
			return fmt.Errorf("force 会修改迁移版本状态，必须显式指定 -yes")
		}
		err = m.Force(opts.version)
	default:
		return fmt.Errorf("不支持的迁移动作: %s", opts.action)
	}
	if err != nil {
		return fmt.Errorf("执行迁移 %s 失败: %w", opts.action, err)
	}
	return printStatus(m, sourceDriver)
}

func printStatus(m *migrate.Migrate, sourceDriver source.Driver) error {
	version, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		version, dirty, err = 0, false, nil
	}
	if err != nil {
		return fmt.Errorf("读取迁移版本: %w", err)
	}
	fmt.Printf("current_version=%d dirty=%t\n", version, dirty)
	current, err := sourceDriver.First()
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("读取迁移清单: %w", err)
	}
	for {
		reader, identifier, readErr := sourceDriver.ReadUp(current)
		if readErr != nil {
			return fmt.Errorf("读取迁移版本 %d: %w", current, readErr)
		}
		_ = reader.Close()
		state := "pending"
		if current == version && dirty {
			state = "dirty"
		} else if current <= version {
			state = "applied"
		}
		fmt.Printf("migration=%d state=%s source=%s\n", current, state, identifier)
		next, nextErr := sourceDriver.Next(current)
		if errors.Is(nextErr, os.ErrNotExist) {
			return nil
		}
		if nextErr != nil {
			return fmt.Errorf("读取下一迁移版本: %w", nextErr)
		}
		current = next
	}
}

func fileURL(dir string) string {
	path := filepath.ToSlash(dir)
	if filepath.IsAbs(dir) {
		return "file://" + path
	}
	return "file://" + path
}

func mysqlURL(dsn string) string {
	return "mysql://" + dsn
}
