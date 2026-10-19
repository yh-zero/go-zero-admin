package config

import (
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type Config struct {
	rest.RestConf
	AppProxyTrustedCIDRs string `json:",optional"`
	JwtAuth              struct {
		AccessSecret string
	}
	AppJwtAuth struct {
		AccessSecret string `json:",optional"`
	} `json:",optional"`
	Page struct {
		PageNo   int64
		PageSize int64
	}
	AppletRPC zrpc.RpcClientConf
	AIRPC     zrpc.RpcClientConf `json:",optional"`
}
