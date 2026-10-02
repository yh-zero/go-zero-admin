// Package rpcprivacy retains method/duration logs while omitting RPC bodies.
package rpcprivacy

import (
	"github.com/zeromicro/go-zero/zrpc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"slices"
)

var credentialMethods = []string{
	"/pb.User/GetUserInfo",    // Login password.
	"/pb.User/Register",       // New-user password.
	"/pb.User/ChangePassword", // Old and new passwords.
	"/pb.User/GetUserToke",    // Client slow-call logs also include the token response.
}

// Discover linked protobuf services so newly added methods are private by
// default. Request bodies, personal profiles, audit snapshots and signed file
// URLs must not accidentally appear in slow-call or error logs.
func privateMethods() []string {
	methods := slices.Clone(credentialMethods)
	protoregistry.GlobalFiles.RangeFiles(func(file protoreflect.FileDescriptor) bool {
		services := file.Services()
		for i := 0; i < services.Len(); i++ {
			service := services.Get(i)
			for j := 0; j < service.Methods().Len(); j++ {
				method := "/" + string(service.FullName()) + "/" + string(service.Methods().Get(j).Name())
				if !slices.Contains(methods, method) {
					methods = append(methods, method)
				}
			}
		}
		return true
	})
	return methods
}

// ConfigureClient retains error/duration logs while omitting request/reply bodies.
func ConfigureClient() {
	for _, method := range privateMethods() {
		zrpc.DontLogClientContentForMethod(method)
	}
}

// ConfigureServer retains configured methods and the framework's timing metrics.
func ConfigureServer(config *zrpc.RpcServerConf) {
	for _, method := range privateMethods() {
		if !slices.Contains(config.Middlewares.StatConf.IgnoreContentMethods, method) {
			config.Middlewares.StatConf.IgnoreContentMethods = append(config.Middlewares.StatConf.IgnoreContentMethods, method)
		}
	}
}
