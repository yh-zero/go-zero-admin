package xerr

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/status"
)

// Handler httpx.SetErrorHandlerCtx 的回调：统一错误信封（契约：一律 HTTP 200 + code/msg）
func Handler(ctx context.Context, err error) (int, interface{}) {
	// 1. 进程内业务错误
	if ce, ok := AsCodeError(err); ok {
		return http.StatusOK, map[string]interface{}{"code": ce.GetErrCode(), "msg": ce.GetErrMsg()}
	}
	// 2. RPC 错误：用 status details 还原业务码（否则退化为 Unknown=2）
	if st, ok := status.FromError(err); ok {
		if code, msg, found := FromGRPCError(err); found {
			return http.StatusOK, map[string]interface{}{"code": code, "msg": msg}
		}
		// 非业务 gRPC 错误（对端不可用、超时等）统一 505，避免传输层数字码泄漏进业务码空间
		return http.StatusOK, map[string]interface{}{"code": uint32(SERVER_COMMON_ERROR), "msg": st.Message()}
	}
	// 3. 参数解析等框架错误：同样走信封，不再返回裸 HTTP 400
	return http.StatusOK, map[string]interface{}{"code": uint32(REUQEST_PARAM_ERROR), "msg": err.Error()}
}

// Unauthorized JWT 失败回调
func Unauthorized(w http.ResponseWriter, r *http.Request, err error) {
	httpx.ErrorCtx(r.Context(), w, NewErrCode(TOKEN_EXPIRE_ERROR))
}
