// sessioncleanup is an explicitly invoked maintenance tool. It never schedules
// itself and opens only the configured database, without starting RPC or Redis.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zeromicro/go-zero/core/conf"
	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/sessioncleanup"
	"os"
	"os/signal"
)

func main() {
	configPath := flag.String("f", "application/applet/rpc/etc/applet.yaml", "applet RPC configuration file")
	retention := flag.Int("retention-days", 0, "required retention days (1..36500) after expiry/revocation")
	batch := flag.Int("batch-size", 0, "required rows per transaction (1..5000)")
	apply := flag.Bool("apply", false, "delete eligible rows; default is dry-run")
	flag.Parse()
	if *retention < 1 || *retention > 36500 || *batch < 1 || *batch > 5000 {
		fmt.Fprintln(os.Stderr, "explicit -retention-days (1..36500) and -batch-size (1..5000) are required")
		os.Exit(2)
	}
	var c config.Config
	if err := conf.Load(*configPath, &c, conf.UseEnv()); err != nil {
		fmt.Fprintln(os.Stderr, "unable to load maintenance configuration")
		os.Exit(1)
	}
	db, err := orm.NewMysql(&orm.Config{DSN: c.DB.DataSource, MaxOpenConns: 2, MaxIdleCnns: 1, MaxLifetime: 60})
	if err != nil {
		fmt.Fprintln(os.Stderr, "unable to open maintenance database")
		os.Exit(1)
	}
	sqlDB, _ := db.DB.DB()
	defer sqlDB.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	result, err := sessioncleanup.Run(ctx, db.DB, sessioncleanup.Options{RetentionDays: *retention, BatchSize: *batch, Apply: *apply})
	_ = json.NewEncoder(os.Stdout).Encode(result)
	if err != nil {
		fmt.Fprintln(os.Stderr, "cleanup failed; earlier completed batches remain committed")
		os.Exit(1)
	}
}
