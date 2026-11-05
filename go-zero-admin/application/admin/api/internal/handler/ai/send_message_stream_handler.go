// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2
// 本入口响应为 SSE 流式（text/event-stream），不返回统一 JSON 信封；
// 信封错误仅在开流前（解析/校验/建立 RPC 流失败）返回。

package ai

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/admin/api/internal/logic/ai"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

// sseHeartbeat 小于 nginx proxy_read_timeout(125s)，防止空闲流被代理掐断。
const sseHeartbeat = 15 * time.Second

func SendMessageStreamHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AiSendMessageReq
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := ai.NewSendMessageStreamLogic(r.Context(), svcCtx)
		stream, err := l.SendMessageStream(&req)
		if err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		header := w.Header()
		header.Set("Content-Type", "text/event-stream; charset=utf-8")
		header.Set("Cache-Control", "no-cache")
		header.Set("X-Accel-Buffering", "no") // 关闭 nginx 代理缓冲
		rc := http.NewResponseController(w)   // 穿透中间件 writer 包装链定位 Flusher
		if err := rc.Flush(); err != nil {
			return
		}

		type recvResult struct {
			chunk *aipb.StreamChunk
			err   error
		}
		// stream 以 r.Context() 建立：客户端断开时 ctx 取消，Recv 退出，goroutine 不泄漏。
		recvCh := make(chan recvResult, 16)
		go func() {
			defer close(recvCh)
			for {
				chunk, err := stream.Recv()
				if err != nil {
					if err != io.EOF {
						recvCh <- recvResult{err: err}
					}
					return
				}
				recvCh <- recvResult{chunk: chunk}
			}
		}()

		ticker := time.NewTicker(sseHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				if _, err := io.WriteString(w, ": ping\n\n"); err != nil {
					return
				}
				_ = rc.Flush()
			case result, ok := <-recvCh:
				if !ok {
					return
				}
				if result.err != nil {
					writeSSEEvent(w, "error", map[string]any{"code": 500, "msg": "流式通道中断，请查询会话后重试"})
					_ = rc.Flush()
					return
				}
				terminal := writeStreamChunk(w, result.chunk)
				_ = rc.Flush()
				if terminal {
					return
				}
			}
		}
	}
}

// writeStreamChunk 将 RPC 帧映射为 SSE 事件；返回 true 表示终帧（done/error）。
func writeStreamChunk(w io.Writer, chunk *aipb.StreamChunk) bool {
	switch chunk.Type {
	case "meta":
		writeSSEEvent(w, "meta", map[string]any{"conversation_id": chunk.ConversationId, "user_message_id": chunk.UserMessageId, "ai_message_id": chunk.AiMessageId})
	case "delta":
		writeSSEEvent(w, "delta", map[string]any{"content": chunk.Content})
	case "tool":
		writeSSEEvent(w, "tool", map[string]any{"name": chunk.Content, "status": chunk.ToolStatus})
	case "done":
		writeSSEEvent(w, "done", map[string]any{"tokens": chunk.Tokens, "usage_complete": chunk.UsageComplete})
		return true
	case "error":
		writeSSEEvent(w, "error", map[string]any{"code": chunk.ErrorCode, "msg": chunk.Content})
		return true
	}
	return false
}

func writeSSEEvent(w io.Writer, event string, data map[string]any) {
	payload, err := json.Marshal(data)
	if err != nil {
		return
	}
	_, _ = io.WriteString(w, "event: "+event+"\ndata: "+string(payload)+"\n\n")
}
