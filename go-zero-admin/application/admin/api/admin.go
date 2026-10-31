// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"strings"

	"go-zero-admin/application/admin/api/internal/config"
	"go-zero-admin/application/admin/api/internal/handler"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/conf"
	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/rest/httpx"
)

var configFile = flag.String("f", "etc/admin.yaml", "the config file")

// 公开路由（不需要鉴权）
var publicPaths = map[string]bool{
	"/ping":                true,
	"/ready":               true,
	"/v1/sys/auth/refresh": true,
	"/v1/sys/auth/captcha": true,
	"/v1/sys/auth/login":   true,
}

func main() {
	flag.Parse()

	var c config.Config
	conf.MustLoad(*configFile, &c)
	// 官方 HTTP 日志在错误分支会完整转储请求，认证接口改用不采集内容的审计与指标。
	c.Middlewares.Log = false

	// 统一错误处理（go-zero 官方推荐入口）
	httpx.SetErrorHandlerCtx(func(ctx context.Context, err error) (int, any) {
		middleware.MarkAuditError(ctx)
		return xerr.Handler(ctx, err)
	})
	httpx.SetOkHandler(xerr.Success)

	server := rest.MustNewServer(c.RestConf, rest.WithUnauthorizedCallback(func(w http.ResponseWriter, r *http.Request, err error) {
		httpx.ErrorCtx(r.Context(), w, xerr.NewErrCode(xerr.TOKEN_EXPIRE_ERROR))
	}))
	defer server.Stop()

	ctx := svc.NewServiceContext(c)
	handler.RegisterHandlers(server, ctx)
	for _, route := range server.Routes() {
		if !strings.HasPrefix(route.Path, "/v1/") {
			continue
		}
		parts := strings.Split(strings.Trim(route.Path, "/"), "/")
		group := strings.Join(parts[1:min(3, len(parts))], "/")
		ctx.ApiRoutes = append(ctx.ApiRoutes, &adminpb.ApiRoute{Method: route.Method, Path: route.Path, Group: group, Title: route.Method + " " + route.Path})
	}

	// 请求元数据（真实 IP/UA 供登录限流与溯源）
	server.Use(middleware.RequestMeta)
	server.Use(middleware.Audit(ctx.AuditRpc))

	// Casbin Authority 全局中间件：deny-by-default，仅 publicPaths 白名单放行
	authority := middleware.NewAuthority(ctx.Enforcer, c.Auth.AccessSecret, ctx.Sessions, ctx.ProfileRpc, ctx.DeptRpc)
	appAuth := middleware.NewAppAuthMiddleware(ctx)
	server.Use(func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/v1/app/") {
				appAuth.Handle(next)(w, r)
				return
			}
			if publicPaths[r.URL.Path] {
				next(w, r)
				return
			}
			authority.Handle(next)(w, r)
		}
	})

	fmt.Printf("Starting server at %s:%d...\n", c.Host, c.Port)
	server.Start()
}
