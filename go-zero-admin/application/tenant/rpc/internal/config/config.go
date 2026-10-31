package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql      struct{ DataSource string }
	CacheRedis cache.CacheConf

	Tenant struct {
		StackTemplateDir  string `json:",default=docker/tenant-stack"`
		NginxConfDir      string `json:",default=docker/nginx/conf.d"`
		EdgeNetwork       string `json:",default=gozero-edge"`
		DomainSuffix      string `json:",default=localhost"`
		StateDir          string `json:",default=docker/tenant-stack/state"`
		InitSQLPath       string `json:",default=data/db/init/mysql-all.sql"`
		NginxContainer    string `json:",default=gozero-platform-nginx-1"`
		StorageConfigPath string `json:",default=docker/etc/file-rpc.yaml"`
	}
}
