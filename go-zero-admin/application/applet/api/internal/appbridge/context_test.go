package appbridge

import (
	"net/http/httptest"
	"testing"
)

func TestRequestIdentityIgnoresSpoofedForwarding(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/app/auth/login", nil)
	r.RemoteAddr = "127.0.0.8:54321"
	r.Header.Set("X-Forwarded-For", "203.0.113.8")
	r.Header.Set("Authorization", "Bearer app-token")
	r.Header.Set("User-Agent", "app-test")
	meta := Request(WithRequest(r).Context())
	if meta.IP != "127.0.0.8" || meta.Token != "app-token" || meta.UserAgent != "app-test" {
		t.Fatalf("unexpected trusted metadata: %+v", meta)
	}
	r.Header.Set("Authorization", "Basic app-token")
	if Request(WithRequest(r).Context()).Token != "" {
		t.Fatal("accepted non-bearer authorization")
	}
}

func TestRequestIdentityRejectsInvalidRemoteAddress(t *testing.T) {
	for _, remote := range []string{"not-an-ip:1234", "not-an-ip", "", "[fe80::1%eth0]:1234"} {
		t.Run(remote, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/v1/app/auth/login", nil)
			r.RemoteAddr = remote
			r.Header.Set("X-Forwarded-For", "203.0.113.8")
			if got := Request(WithRequest(r).Context()).IP; got != "" {
				t.Fatalf("invalid peer address became client IP: %q", got)
			}
		})
	}
}

func TestClientIPResolverTrustBoundary(t *testing.T) {
	resolver, err := NewClientIPResolver("10.0.0.0/8, 2001:db8:ffff::/48")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, remote, forwarded, real, want string
	}{
		{name: "direct spoofing ignored", remote: "198.51.100.9:1234", forwarded: "203.0.113.8", real: "203.0.113.9", want: "198.51.100.9"},
		{name: "first proxied client", remote: "10.0.0.2:1234", forwarded: "203.0.113.8", want: "203.0.113.8"},
		{name: "second proxied client", remote: "10.0.0.2:1234", forwarded: "203.0.113.9", want: "203.0.113.9"},
		{name: "trusted chain peeled from right", remote: "10.0.0.2:1234", forwarded: "192.0.2.99, 198.51.100.7, 10.0.0.3", want: "198.51.100.7"},
		{name: "untrusted intermediary stops peeling", remote: "10.0.0.2:1234", forwarded: "203.0.113.8, 198.51.100.7", want: "198.51.100.7"},
		{name: "untrusted prefix is irrelevant", remote: "10.0.0.2:1234", forwarded: "garbage, 198.51.100.7, 10.0.0.3", want: "198.51.100.7"},
		{name: "all trusted uses leftmost", remote: "10.0.0.2:1234", forwarded: "10.0.0.4, 10.0.0.3", want: "10.0.0.4"},
		{name: "trusted real IP fallback", remote: "10.0.0.2:1234", real: "203.0.113.8", want: "203.0.113.8"},
		{name: "forwarded preferred over real", remote: "10.0.0.2:1234", forwarded: "198.51.100.7", real: "203.0.113.8", want: "198.51.100.7"},
		{name: "malformed hop fails closed", remote: "10.0.0.2:1234", forwarded: "203.0.113.8, garbage, 10.0.0.3", real: "192.0.2.99", want: "10.0.0.2"},
		{name: "empty hop fails closed", remote: "10.0.0.2:1234", forwarded: "203.0.113.8, , 10.0.0.3", want: "10.0.0.2"},
		{name: "forwarded port rejected", remote: "10.0.0.2:1234", forwarded: "203.0.113.8:1234", want: "10.0.0.2"},
		{name: "invalid real IP ignored", remote: "10.0.0.2:1234", real: "not-an-ip", want: "10.0.0.2"},
		{name: "real IP list rejected", remote: "10.0.0.2:1234", real: "203.0.113.8, 198.51.100.7", want: "10.0.0.2"},
		{name: "invalid peer cannot establish trust", remote: "not-an-ip:1234", forwarded: "203.0.113.8", real: "203.0.113.9", want: ""},
		{name: "IPv6 peer and client", remote: "[2001:db8:ffff::1]:1234", forwarded: "2001:db8:1::8, 2001:db8:ffff::2", want: "2001:db8:1::8"},
		{name: "IPv4 mapped peer normalized", remote: "[::ffff:10.0.0.2]:1234", forwarded: "::ffff:203.0.113.8", want: "203.0.113.8"},
		{name: "bare peer address", remote: "10.0.0.2", forwarded: "203.0.113.8", want: "203.0.113.8"},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/v1/app/auth/login", nil)
			r.RemoteAddr = test.remote
			if test.forwarded != "" {
				r.Header.Set("X-Forwarded-For", test.forwarded)
			}
			if test.real != "" {
				r.Header.Set("X-Real-IP", test.real)
			}
			r.Header.Set("Authorization", "Bearer app-token")
			r.Header.Set("User-Agent", "app-test")
			meta := Request(resolver.WithRequest(r).Context())
			if meta.IP != test.want || meta.Token != "app-token" || meta.UserAgent != "app-test" {
				t.Fatalf("metadata=%+v, want IP=%q and original token/user agent", meta, test.want)
			}
		})
	}
}

func TestClientIPResolverRepeatedForwardingHeaders(t *testing.T) {
	resolver, err := NewClientIPResolver("10.0.0.0/8")
	if err != nil {
		t.Fatal(err)
	}
	r := httptest.NewRequest("POST", "/v1/app/auth/login", nil)
	r.RemoteAddr = "10.0.0.2:1234"
	r.Header.Add("X-Forwarded-For", "203.0.113.8")
	r.Header.Add("X-Forwarded-For", "198.51.100.7, 10.0.0.3")
	if got := Request(resolver.WithRequest(r).Context()).IP; got != "198.51.100.7" {
		t.Fatalf("did not respect the last forwarded header: %q", got)
	}
	r.Header.Del("X-Forwarded-For")
	r.Header.Add("X-Real-IP", "203.0.113.8")
	r.Header.Add("X-Real-IP", "198.51.100.7")
	if got := Request(resolver.WithRequest(r).Context()).IP; got != "10.0.0.2" {
		t.Fatalf("accepted ambiguous real IP headers: %q", got)
	}
}

func TestClientIPResolverConfiguration(t *testing.T) {
	for _, cidrs := range []string{"", "  ", "none", " NONE "} {
		t.Run("disabled "+cidrs, func(t *testing.T) {
			resolver, err := NewClientIPResolver(cidrs)
			if err != nil {
				t.Fatal(err)
			}
			r := httptest.NewRequest("POST", "/v1/app/auth/login", nil)
			r.RemoteAddr = "10.0.0.2:1234"
			r.Header.Set("X-Forwarded-For", "203.0.113.8")
			r.Header.Set("X-Real-IP", "203.0.113.9")
			if got := Request(resolver.WithRequest(r).Context()).IP; got != "10.0.0.2" {
				t.Fatalf("disabled proxy trust accepted a forwarded IP: %q", got)
			}
		})
	}
	for _, cidrs := range []string{"not-a-cidr", "10.0.0.1", "10.0.0.0/33", "2001:db8::/129", "10.0.0.0/8,", ",10.0.0.0/8", "none,10.0.0.0/8", "10.0.0.0/8,invalid"} {
		t.Run("invalid "+cidrs, func(t *testing.T) {
			if _, err := NewClientIPResolver(cidrs); err == nil {
				t.Fatal("invalid trusted proxy configuration accepted")
			}
		})
	}
}
