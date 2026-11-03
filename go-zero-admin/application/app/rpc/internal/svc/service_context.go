package svc

import (
	"crypto/tls"
	goredis "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
	"github.com/zeromicro/go-zero/zrpc"

	"go-zero-admin/application/app/rpc/internal/config"
	"go-zero-admin/application/app/rpc/internal/model"
	"go-zero-admin/application/file/rpc/client/fileservice"
	"go-zero-admin/application/file/rpc/filepb"
	"go-zero-admin/pkg/jwtx"
)

type ServiceContext struct {
	Config   config.Config
	DB       sqlx.SqlConn
	Redis    *redis.Redis
	Sessions *jwtx.SessionStore
	FileRpc  fileservice.FileService

	AppUserModel            model.AppUserModel
	AppMembershipGrantModel model.AppMembershipGrantModel
	AppFeedbackModel        model.AppFeedbackModel
}

func NewServiceContext(c config.Config) *ServiceContext {
	if c.SMTP.From == "" {
		c.SMTP.From = c.SMTP.Username
	}
	conn := sqlx.NewMysql(c.Mysql.DataSource)
	rdb := redis.MustNewRedis(c.CacheRedis[0].RedisConf)
	opts := &goredis.Options{Addr: c.CacheRedis[0].Host, Username: c.CacheRedis[0].User, Password: c.CacheRedis[0].Pass}
	if c.CacheRedis[0].Tls {
		opts.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	ctx := &ServiceContext{
		Config:                  c,
		DB:                      conn,
		Redis:                   rdb,
		Sessions:                jwtx.NewSessionStore(goredis.NewClient(opts), "app"),
		AppUserModel:            model.NewAppUserModel(conn, c.CacheRedis),
		AppMembershipGrantModel: model.NewAppMembershipGrantModel(conn, c.CacheRedis),
		AppFeedbackModel:        model.NewAppFeedbackModel(conn, c.CacheRedis),
	}
	if c.FileRpc != nil {
		desc := filepb.File_application_file_rpc_desc_file_proto.Services().Get(0)
		for i := 0; i < desc.Methods().Len(); i++ {
			zrpc.DontLogClientContentForMethod("/" + string(desc.FullName()) + "/" + string(desc.Methods().Get(i).Name()))
		}
		ctx.FileRpc = fileservice.NewFileService(zrpc.MustNewClient(*c.FileRpc))
	}
	return ctx
}
