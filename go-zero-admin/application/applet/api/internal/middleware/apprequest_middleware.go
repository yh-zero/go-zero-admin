// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package middleware

import "net/http"

type AppRequestMiddleware struct {
}

func NewAppRequestMiddleware() *AppRequestMiddleware {
	return &AppRequestMiddleware{}
}

func (m *AppRequestMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		AppRequest(next)(w, r)
	}
}
