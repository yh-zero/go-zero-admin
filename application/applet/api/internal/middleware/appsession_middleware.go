// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import (
	"go-zero-admin/application/applet/rpc/client/appaccountservice"
	"net/http"
)

type AppSessionMiddleware struct {
	client appaccountservice.AppAccountService
}

func NewAppSessionMiddleware(client appaccountservice.AppAccountService) *AppSessionMiddleware {
	return &AppSessionMiddleware{client: client}
}

func (m *AppSessionMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return AppSession(m.client)(next)
}
