package xerr

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/status"
)

// Success 由 go-zero 成功出口统一包装，业务 logic 不处理 HTTP 响应。
func Success(_ context.Context, data any) any {
	return map[string]any{"code": SUCCESS, "msg": MapErrMsg(SUCCESS), "result": data}
}

// Handler 业务错误保持 HTTP 200；就绪探针失败使用 HTTP 503。
func Handler(ctx context.Context, err error) (int, interface{}) {
	// 1. 进程内业务错误
	if ce, ok := AsCodeError(err); ok {
		if ce.GetErrCode() == SERVICE_UNAVAILABLE {
			return http.StatusServiceUnavailable, map[string]any{"code": ce.GetErrCode(), "msg": ce.GetErrMsg()}
		}
		return http.StatusOK, map[string]interface{}{"code": ce.GetErrCode(), "msg": ce.GetErrMsg()}
	}
	// 2. RPC 错误：用 status details 还原业务码（否则退化为 Unknown=2）
	if _, ok := status.FromError(err); ok {
		if code, msg, found := FromGRPCError(err); found {
			return http.StatusOK, map[string]interface{}{"code": code, "msg": msg}
		}
		// 传输错误只返回通用提示，避免暴露内部地址和底层错误细节。
		return http.StatusOK, map[string]interface{}{"code": uint32(SERVER_COMMON_ERROR), "msg": MapErrMsg(SERVER_COMMON_ERROR)}
	}
	// 3. 参数解析等框架错误：同样走信封，不再返回裸 HTTP 400
	return http.StatusOK, map[string]interface{}{"code": uint32(REQUEST_PARAM_ERROR), "msg": MapErrMsg(REQUEST_PARAM_ERROR)}
}

// Unauthorized JWT 失败回调
func Unauthorized(w http.ResponseWriter, r *http.Request, err error) {
	httpx.ErrorCtx(r.Context(), w, NewErrCode(TOKEN_EXPIRE_ERROR))
}
