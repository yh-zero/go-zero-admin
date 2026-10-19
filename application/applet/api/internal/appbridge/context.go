package appbridge

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"

	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/ctxJwt"
)

type requestKey struct{}
type Metadata struct{ IP, UserAgent, Token string }

// ClientIPResolver trusts forwarding headers only from explicitly configured peers.
// Its zero value, including a nil receiver, trusts no proxies.
type ClientIPResolver struct {
	trustedProxies []*net.IPNet
}

// NewClientIPResolver accepts comma-separated CIDRs. Empty or "none" disables
// proxy trust; malformed entries are errors so startup cannot silently weaken it.
func NewClientIPResolver(cidrs string) (*ClientIPResolver, error) {
	resolver := &ClientIPResolver{}
	cidrs = strings.TrimSpace(cidrs)
	if cidrs == "" || strings.EqualFold(cidrs, "none") {
		return resolver, nil
	}
	for _, value := range strings.Split(cidrs, ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(value))
		if err != nil {
			return nil, fmt.Errorf("invalid AppProxyTrustedCIDRs entry %q: %w", value, err)
		}
		resolver.trustedProxies = append(resolver.trustedProxies, network)
	}
	return resolver, nil
}

func (resolver *ClientIPResolver) trusts(ip net.IP) bool {
	if resolver != nil {
		for _, network := range resolver.trustedProxies {
			if network.Contains(ip) {
				return true
			}
		}
	}
	return false
}

func (resolver *ClientIPResolver) clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	if peer == nil {
		return ""
	}
	if !resolver.trusts(peer) {
		return peer.String()
	}
	if forwarded := r.Header.Values("X-Forwarded-For"); len(forwarded) > 0 {
		hops := strings.Split(strings.Join(forwarded, ","), ",")
		for i := len(hops) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(hops[i]))
			if ip == nil {
				// Never skip an invalid hop and trust a client-controlled prefix.
				return peer.String()
			}
			if !resolver.trusts(ip) || i == 0 {
				return ip.String()
			}
		}
	}
	if real := r.Header.Values("X-Real-IP"); len(real) == 1 {
		if ip := net.ParseIP(strings.TrimSpace(real[0])); ip != nil {
			return ip.String()
		}
	}
	return peer.String()
}

// WithRequest uses the connected peer without trusting forwarding headers.
func WithRequest(r *http.Request) *http.Request {
	return (*ClientIPResolver)(nil).WithRequest(r)
}

func (resolver *ClientIPResolver) WithRequest(r *http.Request) *http.Request {
	fields := strings.Fields(r.Header.Get("Authorization"))
	token := ""
	if len(fields) == 2 && strings.EqualFold(fields[0], "Bearer") {
		token = fields[1]
	}
	return r.WithContext(context.WithValue(r.Context(), requestKey{}, Metadata{IP: resolver.clientIP(r), UserAgent: r.UserAgent(), Token: token}))
}
func Request(ctx context.Context) Metadata {
	value, _ := ctx.Value(requestKey{}).(Metadata)
	return value
}
func Actor(ctx context.Context) *pb.SessionRequest {
	data := ctxJwt.GetJwtData(ctx)
	return &pb.SessionRequest{UserID: data.ID, AuthorityId: data.AuthorityId, SessionID: data.SessionID, SessionVersion: data.SessionVersion}
}
