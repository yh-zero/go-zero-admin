package middleware

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
)

// RequestMeta 默认只信任 TCP 对端，供不配置反代的本机入口使用。
func RequestMeta(next http.HandlerFunc) http.HandlerFunc {
	return requestMeta(nil)(next)
}

// NewRequestMeta 在启动时验证显式信任范围，错误配置不能退化为信任所有来源。
func NewRequestMeta(values []string) (func(http.HandlerFunc) http.HandlerFunc, error) {
	trusted := make([]netip.Prefix, 0, len(values))
	for i, value := range values {
		value = strings.TrimSpace(value)
		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			addr, addrErr := netip.ParseAddr(value)
			if addrErr != nil || addr.Zone() != "" {
				return nil, fmt.Errorf("TrustedProxies[%d] 必须为 IP 地址或 CIDR", i)
			}
			addr = addr.Unmap()
			prefix = netip.PrefixFrom(addr, addr.BitLen())
		}
		if prefix.Addr().Is4In6() && prefix.Bits() >= 96 {
			prefix = netip.PrefixFrom(prefix.Addr().Unmap(), prefix.Bits()-96)
		}
		if prefix.Bits() == 0 {
			return nil, fmt.Errorf("TrustedProxies[%d] 不能信任全部来源", i)
		}
		trusted = append(trusted, prefix.Masked())
	}
	return requestMeta(trusted), nil
}

func requestMeta(trusted []netip.Prefix) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := context.WithValue(r.Context(), CtxKeyClientIP, clientIP(r, trusted...))
			ctx = context.WithValue(ctx, CtxKeyUserAgent, r.UserAgent())
			next(w, r.WithContext(ctx))
		}
	}
}

func clientIP(r *http.Request, trusted ...netip.Prefix) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer, err := netip.ParseAddr(host)
	if err != nil {
		return host
	}
	peer = peer.Unmap()
	if !trustedProxy(peer, trusted) {
		return peer.String()
	}
	forwarded := strings.Join(r.Header.Values("X-Forwarded-For"), ",")
	if strings.TrimSpace(forwarded) != "" {
		parts := strings.Split(forwarded, ",")
		chain := make([]netip.Addr, len(parts))
		// 先验证整条链，不能因右侧合法而忽略左侧的损坏输入。
		for i, part := range parts {
			addr, parseErr := netip.ParseAddr(strings.TrimSpace(part))
			if parseErr != nil || addr.Zone() != "" {
				return peer.String()
			}
			chain[i] = addr.Unmap()
		}
		for i := len(chain) - 1; i >= 0; i-- {
			if !trustedProxy(chain[i], trusted) || i == 0 {
				return chain[i].String()
			}
		}
	}
	if real := strings.TrimSpace(strings.Join(r.Header.Values("X-Real-IP"), ",")); real != "" {
		addr, parseErr := netip.ParseAddr(real)
		if parseErr == nil && addr.Zone() == "" {
			return addr.Unmap().String()
		}
	}
	return peer.String()
}

func trustedProxy(addr netip.Addr, trusted []netip.Prefix) bool {
	for _, prefix := range trusted {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}
