package agenttools

import "context"

// 工具执行身份：由服务端编排入口在调用 Execute 前注入，工具按此做归属过滤。
// 禁止从工具参数读取用户身份（LLM 输出不可信）。
type actorKey struct{}

// WithActor 将当前操作者用户 ID 注入上下文
func WithActor(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, actorKey{}, userID)
}

// ActorFrom 读取当前操作者用户 ID；未注入时 ok=false，涉及归属的工具必须拒绝执行
func ActorFrom(ctx context.Context) (userID int64, ok bool) {
	v := ctx.Value(actorKey{})
	if v == nil {
		return 0, false
	}
	id, ok := v.(int64)
	return id, ok && id > 0
}
