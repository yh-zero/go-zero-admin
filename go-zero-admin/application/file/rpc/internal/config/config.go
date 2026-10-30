package config

import (
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/zrpc"

	"go-zero-admin/pkg/objectstorage"
)

type Config struct {
	zrpc.RpcServerConf

	Mysql      struct{ DataSource string }
	CacheRedis cache.CacheConf
	Storage    objectstorage.Config
}
