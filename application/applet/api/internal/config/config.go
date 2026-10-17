package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	JwtAuth struct {
		AccessSecret string
	}
	Page     struct {
		PageNo   int64
		PageSize int64
	}
	AppletRPC zrpc.RpcClientConf
	AIRPC     zrpc.RpcClientConf `json:",optional"`
}

