package svc

import (
	"go-zero-admin/application/applet/rpc/internal/config"
	"go-zero-admin/pkg/hash"
	"go-zero-admin/pkg/middlecasbin"
	"go-zero-admin/pkg/orm"
	"go-zero-admin/pkg/rpcsecurity"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/casbin/casbin/v2"
	"github.com/zeromicro/go-zero/core/stores/redis"
)

type ServiceContext struct {
	Config     config.Config
	DB         *orm.DB
	BizRedis   *redis.Redis
	Casbin     *casbin.SyncedCachedEnforcer // 常驻enforcer 供casbin鉴权/策略管理
	PolicySync *middlecasbin.PolicySynchronizer
	// OssClient is created only when the OSS config is fully populated.
	// A nil value means "OSS not configured"; the file logic surfaces a
	// friendly 300002 error instead of nil-panicking.
	OssClient *oss.Client
}

func NewServiceContext(c config.Config) *ServiceContext {
	if err := hash.ValidatePassword(c.Default.UserPassword); err != nil {
		panic("invalid Default.UserPassword: " + err.Error())
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

	var oc *oss.Client
	if c.Oss.Endpoint != "" && c.Oss.AccessKeyId != "" && c.Oss.AccessKeySecret != "" && c.Oss.BucketName != "" {
		connect := c.Oss.ConnectTimeout
		if connect <= 0 {
			connect = 1
		}
		rw := c.Oss.ReadWriteTimeout
		if rw <= 0 {
			rw = 3
		}
		oc, err = oss.New(c.Oss.Endpoint, c.Oss.AccessKeyId, c.Oss.AccessKeySecret, oss.Timeout(connect, rw))
		if err != nil {
			panic("invalid Oss config: " + err.Error())
		}
	}

	return &ServiceContext{
		Config:     c,
		DB:         db,
		BizRedis:   rds,
		Casbin:     casb,
		PolicySync: synchronizer,
		OssClient:  oc,
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
