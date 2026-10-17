package main

import (
	"flag"
	"fmt"
	"net/http"

	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/handler"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/pkg/httpprivacy"
	"go-zero-admin/pkg/result"
	"go-zero-admin/pkg/result/xerr"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
)

var configFile = flag.String("f", "etc/applet-api.yaml", "the config file")

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	httpprivacy.ConfigureServer(&c.RestConf)
	ctx := svc.NewServiceContext(c)
	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
		// go-zero runs JWT before server.Use middleware; record this denied path explicitly.
		ctx.Audit(func(w http.ResponseWriter, r *http.Request) {
			result.HttpResult(r, w, "", xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
		})(w, r)
	}))
	defer server.Stop()
	if err := httpprivacy.Install(); err != nil {
		panic(err)
	}

	server.Use(ctx.Audit)
	handler.RegisterHandlers(server, ctx)

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
