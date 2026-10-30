package svc

import (
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	"go-zero-admin/application/file/rpc/internal/config"
	"go-zero-admin/application/file/rpc/internal/model"
	"go-zero-admin/pkg/objectstorage"
)

type ServiceContext struct {
	Config config.Config
	DB     sqlx.SqlConn
	Redis  *redis.Redis
	Store  *objectstorage.Client

	SysFileModel model.SysFileModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)

	var store *objectstorage.Client
	if c.Storage.Endpoint != "" {
		var err error
		store, err = objectstorage.New(c.Storage)
		if err != nil {
			panic("init object storage failed: " + err.Error())
		}
	}

	return &ServiceContext{
		Config:       c,
		DB:           conn,
		Redis:        rdb,
		Store:        store,
		SysFileModel: model.NewSysFileModel(conn, c.CacheRedis),
	}
}
