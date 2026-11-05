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

		w.Header().Set("X-Accel-Buffering", "no") // 关闭 nginx 代理缓冲
		w.WriteHeader(http.StatusOK)
		if err := flushSSE(w); err != nil {
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
						select {
						case recvCh <- recvResult{err: err}:
						case <-r.Context().Done():
						}
					}
					return
				}
				select {
				case recvCh <- recvResult{chunk: chunk}:
				case <-r.Context().Done():
					return
				}
			}
		}()

		ticker := time.NewTicker(sseHeartbeat)
		defer ticker.Stop()
		for {
			select {
			case <-r.Context().Done():
				return
			case <-ticker.C:
				if err := writeSSEFrame(w, ": ping\n\n"); err != nil {
					return
				}
			case result, ok := <-recvCh:
				if !ok {
					return
				}
				if result.err != nil {
					_ = writeSSEEvent(w, "error", map[string]any{"code": 500, "msg": "流式通道中断，请查询会话后重试"})
					return
				}
				terminal, err := writeStreamChunk(w, result.chunk)
				if err != nil || terminal {
					return
				}
			}
		}
	}
}

// writeStreamChunk 将 RPC 帧映射为 SSE 事件；返回 true 表示终帧（done/error）。
func writeStreamChunk(w http.ResponseWriter, chunk *aipb.StreamChunk) (bool, error) {
	switch chunk.Type {
	case "meta":
		return false, writeSSEEvent(w, "meta", map[string]any{"conversation_id": chunk.ConversationId, "user_message_id": chunk.UserMessageId, "ai_message_id": chunk.AiMessageId})
	case "delta":
		return false, writeSSEEvent(w, "delta", map[string]any{"content": chunk.Content})
	case "tool":
		return false, writeSSEEvent(w, "tool", map[string]any{"name": chunk.Content, "status": chunk.ToolStatus})
	case "done":
		return true, writeSSEEvent(w, "done", map[string]any{"tokens": chunk.Tokens, "usage_complete": chunk.UsageComplete, "result": ai.MapSendResult(chunk.Result)})
	case "error":
		return true, writeSSEEvent(w, "error", map[string]any{"code": chunk.ErrorCode, "msg": chunk.Content})
	}
	return false, nil
}

func writeSSEEvent(w http.ResponseWriter, event string, data map[string]any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return writeSSEFrame(w, "event: "+event+"\ndata: "+string(payload)+"\n\n")
}

func writeSSEFrame(w http.ResponseWriter, frame string) error {
	n, err := io.WriteString(w, frame)
	if err != nil {
		return err
	}
	if n != len(frame) {
		return io.ErrShortWrite
	}
	return flushSSE(w)
}

func flushSSE(w http.ResponseWriter) error {
	// go-zero 的透传 Flush 会吞掉底层错误，先穿透包装，保留可报告错误的 FlushError。
	for {
		if _, ok := w.(interface{ FlushError() error }); ok {
			break
		}
		unwrapper, ok := w.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			break
		}
		w = unwrapper.Unwrap()
	}
	return http.NewResponseController(w).Flush()
}
