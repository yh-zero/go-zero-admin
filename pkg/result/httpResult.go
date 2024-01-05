package result

import (
	"fmt"
	"google.golang.org/grpc/status"
	"net/http"

	"go-zero-admin/pkg/result/xerr"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 设置http 返回数据的结构
func HttpResult(r *http.Request, w http.ResponseWriter, resp interface{}, err error) {
	fmt.Println("========== HttpResult", err)
	if err == nil {
		// 成功返回
		r := Success(resp)
		httpx.WriteJson(w, http.StatusOK, r)
	} else {
		//错误返回
		errcode := xerr.SERVER_COMMON_ERROR
		errmsg := "服务器开小差啦，稍后再来试一试"

		causeErr := errors.Cause(err)         // err类型
		fmt.Println("causeErr：", causeErr)    //causeErr： rpc error: code = Unknown desc = 存在重复name，请修改name
		fmt.Printf("causeErr: %T：", causeErr) //causeErr: *status.Error：

		if e, ok := causeErr.(*xerr.CodeError); ok { //自定义错误类型
			//自定义CodeError
			errcode = e.GetErrCode()
			errmsg = e.GetErrMsg()
		} else {
			if gstatus, ok := status.FromError(causeErr); ok { // grpc err错误
				grpcCode := uint32(gstatus.Code())
				if xerr.IsCodeErr(grpcCode) { //区分自定义错误跟系统底层、db等错误，底层、db错误不能返回给前端
					errcode = grpcCode
					errmsg = gstatus.Message()
				} else { // 临时加上
					errcode = grpcCode
					errmsg = "grpc err:" + gstatus.Message()
				}
			}
		}

		logx.WithContext(r.Context()).Errorf("【API-ERR】 : %+v ", err)
		//httpx.WriteJson(w, http.StatusBadRequest, Error(errcode, errmsg))
		httpx.WriteJson(w, http.StatusOK, Error(errcode, errmsg))
	}
}
