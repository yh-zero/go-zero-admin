package middleware

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/zeromicro/go-zero/core/conf"
	"go-zero-admin/application/admin/api/internal/config"
)

func TestRequestMetaTrustedProxyBoundary(t *testing.T) {
	for _, test := range []struct {
		name, remote, forwarded, real, want string
		trusted                             []string
	}{
		{name: "默认忽略伪造来源头", remote: "192.0.2.9:1234", forwarded: "198.51.100.7", real: "198.51.100.8", want: "192.0.2.9"},
		{name: "非代理客户端不能伪造来源", trusted: []string{"172.20.0.3/32"}, remote: "172.20.0.99:1234", forwarded: "198.51.100.7", real: "198.51.100.8", want: "172.20.0.99"},
		{name: "读取可信代理追加的地址", trusted: []string{"172.20.0.3"}, remote: "172.20.0.3:1234", forwarded: "198.51.100.7", want: "198.51.100.7"},
		{name: "跳过不可信客户端伪造的前缀", trusted: []string{"172.20.0.3/32"}, remote: "172.20.0.3:1234", forwarded: "203.0.113.9, 198.51.100.7", want: "198.51.100.7"},
		{name: "多级可信代理从右向左解析", trusted: []string{"172.20.0.3/32", "10.10.0.0/24"}, remote: "172.20.0.3:1234", forwarded: "203.0.113.9, 198.51.100.7, 10.10.0.8", want: "198.51.100.7"},
		{name: "可信代理单地址回退", trusted: []string{"172.20.0.3"}, remote: "172.20.0.3:1234", real: "198.51.100.8", want: "198.51.100.8"},
		{name: "坏链整体回退TCP", trusted: []string{"172.20.0.3"}, remote: "172.20.0.3:1234", forwarded: "not-an-ip, 198.51.100.7", real: "198.51.100.8", want: "172.20.0.3"},
		{name: "空链成员回退TCP", trusted: []string{"172.20.0.3"}, remote: "172.20.0.3:1234", forwarded: "198.51.100.7, ", want: "172.20.0.3"},
		{name: "带端口的来源头回退TCP", trusted: []string{"172.20.0.3"}, remote: "172.20.0.3:1234", real: "198.51.100.8:80", want: "172.20.0.3"},
		{name: "IPv6代理和客户端", trusted: []string{"2001:db8:1::/64"}, remote: "[2001:db8:1::3]:1234", forwarded: "2001:db8:2::7", want: "2001:db8:2::7"},
		{name: "映射IPv4代理", trusted: []string{"172.20.0.3/32"}, remote: "[::ffff:172.20.0.3]:1234", forwarded: "198.51.100.7", want: "198.51.100.7"},
		{name: "无转发头仍取TCP", trusted: []string{"172.20.0.3"}, remote: "172.20.0.3:1234", want: "172.20.0.3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			meta, err := NewRequestMeta(test.trusted)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = test.remote
			r.Header.Set("X-Forwarded-For", test.forwarded)
			r.Header.Set("X-Real-IP", test.real)
			r.Header.Set("User-Agent", "request-meta-test")
			meta(func(_ http.ResponseWriter, r *http.Request) {
				if got := r.Context().Value(CtxKeyClientIP); got != test.want {
					t.Fatalf("客户端IP=%v，期望 %s", got, test.want)
				}
				if got := r.Context().Value(CtxKeyUserAgent); got != "request-meta-test" {
					t.Fatalf("User-Agent=%v", got)
				}
			})(httptest.NewRecorder(), r)
		})
	}
}

func TestRequestMetaRejectsInvalidTrustConfiguration(t *testing.T) {
	for _, value := range []string{"", "proxy.internal", "172.20.0.3:80", "172.20.0.0/33", "0.0.0.0/0", "::/0"} {
		t.Run(value, func(t *testing.T) {
			if _, err := NewRequestMeta([]string{value}); err == nil {
				t.Fatal("无效或全来源信任配置必须拒绝")
			}
		})
	}
}

func TestRequestMetaLoadsExplicitProxyConfiguration(t *testing.T) {
	root := filepath.Join("..", "..", "..", "..", "..")
	for _, test := range []struct {
		name, path, remote, want string
	}{
		{name: "宿主默认无信任", path: "application/admin/api/etc/admin.yaml", remote: "172.20.0.3:80", want: "172.20.0.3"},
		{name: "Compose精确代理地址随环境覆盖", path: "docker/etc/admin-api.yaml", remote: "172.28.0.3:80", want: "198.51.100.7"},
		{name: "Compose同网段其他来源不可信", path: "docker/etc/admin-api.yaml", remote: "172.28.0.9:80", want: "172.28.0.9"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("GOZERO_PLATFORM_PROXY_IP", "172.28.0.3")
			var c config.Config
			if err := conf.Load(filepath.Join(root, test.path), &c, conf.UseEnv()); err != nil {
				t.Fatal(err)
			}
			meta, err := NewRequestMeta(c.TrustedProxies)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = test.remote
			r.Header.Set("X-Forwarded-For", "198.51.100.7")
			meta(func(_ http.ResponseWriter, r *http.Request) {
				if got := r.Context().Value(CtxKeyClientIP); got != test.want {
					t.Fatalf("已加载配置产生客户端IP=%v，期望=%s", got, test.want)
				}
			})(httptest.NewRecorder(), r)
		})
	}
}
