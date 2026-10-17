package svc

import (
	agentRPC "go-zero-admin/application/ai/rpc/client/agent"
	"go-zero-admin/application/applet/api/internal/config"
	"go-zero-admin/application/applet/api/internal/middleware"
	"go-zero-admin/application/applet/rpc/client/api"
	auditRPC "go-zero-admin/application/applet/rpc/client/audit"
	"go-zero-admin/application/applet/rpc/client/authority"
	captchaRPC "go-zero-admin/application/applet/rpc/client/captcha"
	casbinRPC "go-zero-admin/application/applet/rpc/client/casbin"
	dictionaryRPC "go-zero-admin/application/applet/rpc/client/dictionary"
	"go-zero-admin/application/applet/rpc/client/fileresourceservice"
	mailRPC "go-zero-admin/application/applet/rpc/client/mail"
	"go-zero-admin/application/applet/rpc/client/menu"
	"go-zero-admin/application/applet/rpc/client/organization"
	"go-zero-admin/application/applet/rpc/client/permission"
	"go-zero-admin/application/applet/rpc/client/sessionmanage"
	"go-zero-admin/application/applet/rpc/client/user"
	"go-zero-admin/pkg/audit"
	"go-zero-admin/pkg/rpcprivacy"
	"go-zero-admin/pkg/rpcsecurity"
	"net/http"

	"github.com/zeromicro/go-zero/rest"
	"github.com/zeromicro/go-zero/zrpc"
)

type ServiceContext struct {
	AppletPermissionRPC    permission.Permission
	AIAgentRPC             agentRPC.Agent
	Audit                  rest.Middleware
	AppletAuditRPC         auditRPC.Audit
	AppletOrganizationRPC  organization.Organization
	AppletFileRPC          fileresourceservice.FileResourceService
	AppletMailRPC          mailRPC.Mail
	AppletCaptchaRPC       captchaRPC.Captcha
	AppletSessionManageRPC sessionmanage.SessionManage
	Session                rest.Middleware
	Config                 config.Config
	Authority              rest.Middleware
	AppletUserRPC          user.User
	AppletMenuRPC          menu.Menu
	AppletAuthorityRPC     authority.Authority
	AppletAPIRPC           api.Api
	AppletCasbinRPC        casbinRPC.Casbin
	AppletDictionaryRPC    dictionaryRPC.Dictionary
}

func NewServiceContext(c config.Config) *ServiceContext {
	if err := rpcsecurity.ValidateClient(c.AppletRPC, c.Mode); err != nil {
		panic(err)
	}

	rpcprivacy.ConfigureClient()
	appletRPC := zrpc.MustNewClient(c.AppletRPC, zrpc.WithUnaryClientInterceptor(audit.ClientInterceptor()))
	casbinCli := casbinRPC.NewCasbin(appletRPC)

	svc := &ServiceContext{
		AppletPermissionRPC:    permission.NewPermission(appletRPC),
		AIAgentRPC:             newAIAgentClient(c.AIRPC, c.Mode),
		AppletAuditRPC:         auditRPC.NewAudit(appletRPC),
		AppletOrganizationRPC:  organization.NewOrganization(appletRPC),
		AppletFileRPC:          fileresourceservice.NewFileResourceService(appletRPC),
		AppletMailRPC:          mailRPC.NewMail(appletRPC),
		AppletCaptchaRPC:       captchaRPC.NewCaptcha(appletRPC),
		AppletSessionManageRPC: sessionmanage.NewSessionManage(appletRPC),
		Config:                 c,
		AppletUserRPC:          user.NewUser(appletRPC),
		AppletMenuRPC:          menu.NewMenu(appletRPC),
		AppletAuthorityRPC:     authority.NewAuthority(appletRPC),
		AppletAPIRPC:           api.NewApi(appletRPC),
		AppletCasbinRPC:        casbinCli,
		AppletDictionaryRPC:    dictionaryRPC.NewDictionary(appletRPC),
	}

	// 权限鉴权走RPC casbin api层不直连数据库
	svc.Audit = middleware.NewAuditMiddleware(svc.AppletAuditRPC).Handle
	svc.Session = middleware.NewSessionMiddleware(svc.AppletUserRPC).Handle
	authority := middleware.NewAuthorityMiddleware(casbinCli).Handle
	svc.Authority = func(next http.HandlerFunc) http.HandlerFunc { return svc.Session(authority(next)) }

	return svc
}

