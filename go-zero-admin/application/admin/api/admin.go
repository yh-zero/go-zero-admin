// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"flag"
	"fmt"
	"net/http"
	"strings"

	"go-zero-admin/application/admin/api/internal/config"
	"go-zero-admin/application/admin/api/internal/handler"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/admin.yaml", "the config file")

// 公开路由（不需要鉴权）
var publicPaths = map[string]bool{
	"/ping":                true,
	"/v1/sys/auth/captcha": true,
	"/v1/sys/auth/login":   true,
}

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)

	// 统一错误处理（go-zero 官方推荐入口）
	httpx.SetErrorHandlerCtx(xerr.Handler)

	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
		httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
	}))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)

	// Casbin Authority 全局中间件（公开路由放行）
	authority := middleware.NewAuthority(ctx.Enforcer, c.Auth.AccessSecret)
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if publicPaths[r.URL.Path] {
				next(w, r)
				return
			}
			// 带 Bearer token 的走鉴权，否则也放行（让具体接口决定）
			if strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") {
				authority.Handle(next)(w, r)
				return
			}
			next(w, r)
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
