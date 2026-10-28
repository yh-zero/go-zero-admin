package rpcsecurity

import (
	"context"
	"crypto/subtle"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"

	redisclient "github.com/redis/go-redis/v9"
	"github.com/zeromicro/go-zero/core/service"
	"github.com/zeromicro/go-zero/zrpc"
)

type Credential struct {
	App   string `json:",optional"`
	Token string `json:",optional"`
}

var validApp = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func ValidateCredential(credential Credential) error {
	if !validApp.MatchString(credential.App) {
		return errors.New("RPCAuth.App must contain 1-128 letters, numbers, underscores or hyphens")
	}
	if len(credential.Token) < 32 || strings.ContainsAny(credential.Token, "\r\n") || strings.HasPrefix(credential.Token, "replace-with-") {
		return errors.New("RPCAuth.Token must be a single-line secret of at least 32 bytes; replace template values before startup")
	}
	return nil
}

func ValidateServer(config zrpc.RpcServerConf, credential Credential) error {
	host, _, err := net.SplitHostPort(config.ListenOn)
	if err != nil {
		return fmt.Errorf("invalid RPC ListenOn: %w", err)
	}
	address := net.ParseIP(host)
	loopback := host == "localhost" || address != nil && address.IsLoopback()
	if !config.Auth {
		if config.Mode != service.DevMode && config.Mode != service.TestMode && config.Mode != service.RtMode {
			return errors.New("production RPC requires Auth=true and StrictControl=true, including loopback listeners")
		}
		if !loopback {
			return errors.New("RPC listening outside loopback requires Auth=true and StrictControl=true")
		}
		return nil
	}
	if !config.StrictControl {
		return errors.New("RPC authentication requires StrictControl=true to reject calls when Redis is unavailable")
	}
	if err := config.Validate(); err != nil {
		return err
	}
	return ValidateCredential(credential)
}

func ValidateClient(config zrpc.RpcClientConf, mode string) error {
	if config.App == "" && config.Token == "" {
		if mode == service.DevMode || mode == service.TestMode || mode == service.RtMode {
			return nil
		}
		return errors.New("production AppletRPC requires App and Token credentials")
	}
	return ValidateCredential(Credential{App: config.App, Token: config.Token})
}

// RegisterCredential creates only an absent token. A stale replica cannot
// silently overwrite a rotated token; conflicting configuration stops startup.
func RegisterCredential(config zrpc.RpcServerConf, credential Credential) error {
	if err := ValidateServer(config, credential); err != nil {
		return err
	}
	if !config.Auth {
		return nil
	}
	// Use a dedicated client without go-zero Redis command logging hooks:
	// HSETNX contains the token and must never be logged on slow/error paths.
	options := &redisclient.UniversalOptions{Addrs: strings.Split(config.Redis.Host, ","), Username: config.Redis.User, Password: config.Redis.Pass, DialTimeout: 3 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second}
	if config.Redis.Tls {
		options.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	store := redisclient.NewUniversalClient(options)
	defer store.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	created, err := store.HSetNX(ctx, config.Redis.Key, credential.App, credential.Token).Result()
	if err != nil {
		return errors.New("RPC credential registration failed: authentication Redis is unavailable")
	}
	if created {
		return nil
	}
	existing, err := store.HGet(ctx, config.Redis.Key, credential.App).Result()
	if err != nil {
		return errors.New("RPC credential verification failed: authentication Redis is unavailable")
	}
	if subtle.ConstantTimeCompare([]byte(existing), []byte(credential.Token)) != 1 {
		return errors.New("RPC token conflicts with the registered application token; coordinate token rotation before restarting replicas")
	}
	return nil
}
