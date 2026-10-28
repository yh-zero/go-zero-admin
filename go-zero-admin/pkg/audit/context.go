package audit

import (
	"context"
	"encoding/base64"
	"strconv"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

type Actor struct {
	ID          int64
	Name        string
	AuthorityID int64
}

type Request struct{ Path, Method, IP, TraceID, UserAgent string }

type stateKey struct{}
type state struct {
	sync.RWMutex
	actor   Actor
	request Request
}

// WithRequest shares state across middleware-derived contexts, allowing the
// verified JWT/session middleware and login handler to set the real operator.
func WithRequest(ctx context.Context, request Request) context.Context {
	return context.WithValue(ctx, stateKey{}, &state{request: request})
}

func SetActor(ctx context.Context, actor Actor) {
	if value, ok := ctx.Value(stateKey{}).(*state); ok {
		value.Lock()
		value.actor = actor
		value.Unlock()
	}
}

func ActorFromContext(ctx context.Context) Actor {
	if value, ok := ctx.Value(stateKey{}).(*state); ok {
		value.RLock()
		defer value.RUnlock()
		return value.actor
	}
	md, _ := metadata.FromIncomingContext(ctx)
	id, _ := strconv.ParseInt(first(md, "x-audit-actor-id"), 10, 64)
	authority, _ := strconv.ParseInt(first(md, "x-audit-authority-id"), 10, 64)
	name, _ := base64.RawURLEncoding.DecodeString(first(md, "x-audit-actor-name"))
	return Actor{ID: id, Name: string(name), AuthorityID: authority}
}

func RequestFromContext(ctx context.Context) Request {
	if value, ok := ctx.Value(stateKey{}).(*state); ok {
		value.RLock()
		defer value.RUnlock()
		return value.request
	}
	md, _ := metadata.FromIncomingContext(ctx)
	userAgent, _ := base64.RawURLEncoding.DecodeString(first(md, "x-audit-user-agent"))
	requestPath, _ := base64.RawURLEncoding.DecodeString(first(md, "x-audit-path"))
	return Request{Path: string(requestPath), Method: first(md, "x-audit-method"), IP: first(md, "x-audit-ip"), TraceID: first(md, "x-audit-trace-id"), UserAgent: string(userAgent)}
}

func first(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

// OutgoingContext is called after the session has verified the actor. Metadata
// comes from trusted server state, never from user-supplied x-audit headers.
func OutgoingContext(ctx context.Context) context.Context {
	actor, request := ActorFromContext(ctx), RequestFromContext(ctx)
	existing, _ := metadata.FromOutgoingContext(ctx)
	md := existing.Copy()
	for key, value := range map[string]string{
		"x-audit-actor-id":     strconv.FormatInt(actor.ID, 10),
		"x-audit-actor-name":   base64.RawURLEncoding.EncodeToString([]byte(actor.Name)),
		"x-audit-authority-id": strconv.FormatInt(actor.AuthorityID, 10),
		"x-audit-path":         base64.RawURLEncoding.EncodeToString([]byte(clean(request.Path, 256))), "x-audit-method": clean(request.Method, 16),
		"x-audit-ip": request.IP, "x-audit-trace-id": request.TraceID,
		"x-audit-user-agent": base64.RawURLEncoding.EncodeToString([]byte(clean(request.UserAgent, 256))),
	} {
		md.Set(key, value)
	}
	return metadata.NewOutgoingContext(ctx, md)
}

func ClientInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply any, conn *grpc.ClientConn, invoke grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		return invoke(OutgoingContext(ctx), method, req, reply, conn, opts...)
	}
}
