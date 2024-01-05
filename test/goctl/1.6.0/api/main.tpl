package main

import (
	"flag"
	"fmt"
	"net/http"
	

	"go-zero-admin/pkg/result"
    "go-zero-admin/pkg/result/xerr"
	{{.importPackages}}
)

var configFile = flag.String("f", "etc/{{.serviceName}}.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
    		result.HttpResult(r, w, "", xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
    	}))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
