package middleware

import (
	"go-zero-admin/application/applet/api/internal/appbridge"
	"go-zero-admin/application/applet/rpc/client/appaccountservice"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result"
	"net/http"
)

func AppRequest(next http.HandlerFunc) http.HandlerFunc {
	return AppRequestWithResolver(nil)(next)
}

func AppRequestWithResolver(resolver *appbridge.ClientIPResolver) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) { next(w, resolver.WithRequest(r)) }
	}
}

// AppSession checks the App identity in RPC; a valid admin JWT is insufficient.
func AppSession(client appaccountservice.AppAccountService) func(http.HandlerFunc) http.HandlerFunc {
	return AppSessionWithResolver(client, nil)
}

func AppSessionWithResolver(client appaccountservice.AppAccountService, resolver *appbridge.ClientIPResolver) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return AppRequestWithResolver(resolver)(func(w http.ResponseWriter, r *http.Request) {
			_, err := client.CheckAppSession(r.Context(), &pb.AppAccessRequest{AccessToken: appbridge.Request(r.Context()).Token})
			if err != nil {
				result.HttpResult(r, w, nil, err)
				return
			}
			next(w, r)
		})
	}
}
