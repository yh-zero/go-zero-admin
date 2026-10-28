package svc

import (
	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/pkg/hash"
	"go-zero-admin/pkg/iplocation"
	"go-zero-admin/pkg/middlecasbin"
	"go-zero-admin/pkg/objectstorage"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/rpcsecurity"

	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	IPLocation interface{ Lookup(string) string }
	Config     config.Config
	DB         *orm.DB
	BizRedis   *redis.Redis
	Casbin     *casbin.SyncedCachedEnforcer // 常驻enforcer 供casbin鉴权/策略管理
	PolicySync *middlecasbin.PolicySynchronizer
	// Nil Storage means the optional file storage has not been configured.
	Storage *objectstorage.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	if err := hash.ValidatePassword(c.Default.UserPassword); err != nil {
		panic("invalid Default.UserPassword: " + err.Error())
	}
	storage, err := objectstorage.New(c.Storage)
	if err != nil {
		panic(err)
	}
	location, err := iplocation.New(c.IPLocation.IPv4Path, c.IPLocation.IPv6Path)
	if err != nil {
		panic("invalid IPLocation config: " + err.Error())
	}
	if err := rpcsecurity.RegisterCredential(c.RpcServerConf, c.RPCAuth); err != nil {
		panic(err)
	}
	db := orm.MustNewMysql(&orm.Config{
		DSN:          c.DB.DataSource,
		MaxOpenConns: c.DB.MaxOpenConns,
		MaxIdleCnns:  c.DB.MaxIdleConns,
		MaxLifetime:  c.DB.MaxLifetime,
	})
	rds := redis.MustNewRedis(redis.RedisConf{
		Host: c.BizRedis.Host,
		Type: c.BizRedis.Type,
		Pass: c.BizRedis.Pass,
	})

	casb, synchronizer, err := c.CasbinConf.NewPolicySynchronizer(db.DB, c.BizRedis)
	if err != nil {
		if sqlDB, dbErr := db.DB.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		panic(err)
	}

	return &ServiceContext{
		IPLocation: location,
		Config:     c,
		DB:         db,
		BizRedis:   rds,
		Casbin:     casb,
		PolicySync: synchronizer,
		Storage:    storage,
	}
}

func (s *ServiceContext) Close() {
	if s.PolicySync != nil {
		s.PolicySync.Close()
	}
	if s.DB != nil {
		if sqlDB, err := s.DB.DB.DB(); err == nil {
			_ = sqlDB.Close()
		}
	}
}
