package result

import (
	"errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/pkg/result/xerr"
	"google.golang.org/grpc/status"
	"net/http"
)

// HttpResult 保持业务响应结构；内部错误只写服务端日志，不公开数据库和RPC细节。
func HttpResult(r *http.Request, w http.ResponseWriter, resp any, err error) {
	if err == nil {
		httpx.WriteJson(w, http.StatusOK, Success(resp))
		return
	}
	code := xerr.SERVER_COMMON_ERROR
	message := xerr.MapErrMsg(code)
	var businessError *xerr.CodeError
	if errors.As(err, &businessError) {
		code = businessError.GetErrCode()
		message = businessError.GetErrMsg()
	} else if rpcStatus, ok := status.FromError(err); ok && xerr.IsCodeErr(uint32(rpcStatus.Code())) {
		code = uint32(rpcStatus.Code())
		message = rpcStatus.Message()
	}
	logx.WithContext(r.Context()).Errorf("API request failed: %+v", err)
	httpx.WriteJson(w, http.StatusOK, Error(code, message))
}
