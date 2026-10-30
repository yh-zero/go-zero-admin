package xerr

import (
	"context"
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"
	"google.golang.org/grpc/status"
)

// Handler httpx.SetErrorHandlerCtx 的回调：统一错误翻译
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
		return http.StatusOK, map[string]interface{}{"code": uint32(st.Code()), "msg": st.Message()}
	}
	// 3. 参数解析等框架错误
	return http.StatusBadRequest, map[string]interface{}{"code": REUQEST_PARAM_ERROR, "msg": err.Error()}
}

// Unauthorized JWT 失败回调
func Unauthorized(w http.ResponseWriter, r *http.Request, err error) {
	httpx.ErrorCtx(r.Context(), w, NewErrCode(TOKEN_EXPIRE_ERROR))
}
