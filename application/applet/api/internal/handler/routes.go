// Code generated by goctl. DO NOT EDIT.
package handler

import (
	"net/http"

	api "go-zero-admin/application/applet/api/internal/handler/api"
	authority "go-zero-admin/application/applet/api/internal/handler/authority"
	casbin "go-zero-admin/application/applet/api/internal/handler/casbin"
	menu "go-zero-admin/application/applet/api/internal/handler/menu"
	user "go-zero-admin/application/applet/api/internal/handler/user"
	usernocasbin "go-zero-admin/application/applet/api/internal/handler/usernocasbin"
	"go-zero-admin/application/applet/api/internal/svc"

	"github.com/zeromicro/go-zero/rest"
)

func RegisterHandlers(server *rest.Server, serverCtx *svc.ServiceContext) {
	server.AddRoutes(
		[]rest.Route{
			{
				Method:  http.MethodPost,
				Path:    "/login",
				Handler: usernocasbin.LoginHandler(serverCtx),
			},
			{
				Method:  http.MethodGet,
				Path:    "/randomImage",
				Handler: usernocasbin.RandomImageHandler(serverCtx),
			},
		},
		rest.WithPrefix("/v1/sys"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Authority},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/getUserList",
					Handler: user.GetUserListHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/register",
					Handler: user.RegisterHandler(serverCtx),
				},
				{
					Method:  http.MethodPut,
					Path:    "/updateUserInfo",
					Handler: user.UpdateUserInfoHandler(serverCtx),
				},
				{
					Method:  http.MethodPut,
					Path:    "/resetUserPassword",
					Handler: user.ResetUserPasswordHandler(serverCtx),
				},
			}...,
		),
		rest.WithPrefix("/v1/sys"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Authority},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/getMenu",
					Handler: menu.GetMenuHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/getMenuList",
					Handler: menu.GetMenuListHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/addBaseMenu",
					Handler: menu.AddBaseMenuHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/getBaseMenuTree",
					Handler: menu.GetBaseMenuTreeHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/getMenuAuthority",
					Handler: menu.GetMenuAuthorityHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/getBaseMenuById",
					Handler: menu.GetBaseMenuByIdHandler(serverCtx),
				},
				{
					Method:  http.MethodPut,
					Path:    "/updateBaseMenu",
					Handler: menu.UpdateBaseMenuHandler(serverCtx),
				},
			}...,
		),
		rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
		rest.WithPrefix("/v1/sys/menu"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Authority},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/getAuthorityList",
					Handler: authority.GetAuthorityListHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/addAuthorityMenu",
					Handler: authority.AddAuthorityMenuHandler(serverCtx),
				},
				{
					Method:  http.MethodPut,
					Path:    "/updateAuthority",
					Handler: authority.UpdateAuthorityHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/createAuthority",
					Handler: authority.CreateAuthorityHandler(serverCtx),
				},
			}...,
		),
		rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
		rest.WithPrefix("/v1/sys/authority"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Authority},
			[]rest.Route{
				{
					Method:  http.MethodGet,
					Path:    "/getApiList",
					Handler: api.GetApiListHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/createApi",
					Handler: api.CreateApiHandler(serverCtx),
				},
				{
					Method:  http.MethodDelete,
					Path:    "/deleteApi",
					Handler: api.DeleteApiHandler(serverCtx),
				},
				{
					Method:  http.MethodGet,
					Path:    "/getAllApiList",
					Handler: api.GetAllApiListHandler(serverCtx),
				},
			}...,
		),
		rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
		rest.WithPrefix("/v1/sys/api"),
	)

	server.AddRoutes(
		rest.WithMiddlewares(
			[]rest.Middleware{serverCtx.Authority},
			[]rest.Route{
				{
					Method:  http.MethodPost,
					Path:    "/getPathByAuthorityId",
					Handler: casbin.GetPathByAuthorityIdHandler(serverCtx),
				},
				{
					Method:  http.MethodPost,
					Path:    "/updateCasbinData",
					Handler: casbin.UpdateCasbinDataHandler(serverCtx),
				},
			}...,
		),
		rest.WithJwt(serverCtx.Config.JwtAuth.AccessSecret),
		rest.WithPrefix("/v1/sys/casbin"),
	)
}
