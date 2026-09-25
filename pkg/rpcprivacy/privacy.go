// Package rpcprivacy uses go-zero's method-level log controls for credentials.
package rpcprivacy

import (
	"github.com/zeromicro/go-zero/zrpc"
	"slices"
)

var credentialMethods = []string{
	"/pb.User/GetUserInfo",    // Login password.
	"/pb.User/Register",       // New-user password.
	"/pb.User/ChangePassword", // Old and new passwords.
	"/pb.User/GetUserToke",    // Client slow-call logs also include the token response.
}

// ConfigureClient retains error/duration logs while omitting request/reply bodies.
func ConfigureClient() {
	for _, method := range credentialMethods {
		zrpc.DontLogClientContentForMethod(method)
	}
}

// ConfigureServer retains configured methods and the framework's timing metrics.
func ConfigureServer(config *zrpc.RpcServerConf) {
	for _, method := range credentialMethods {
		if !slices.Contains(config.Middlewares.StatConf.IgnoreContentMethods, method) {
			config.Middlewares.StatConf.IgnoreContentMethods = append(config.Middlewares.StatConf.IgnoreContentMethods, method)
		}
	}
}
