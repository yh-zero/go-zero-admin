package result

import (
	"time"
)

type ResponseSuccessBean struct {
	Code       uint32      `json:"code"`
	Message    string      `json:"message"`
	Result     interface{} `json:"result"`
	ReturnData interface{} `json:"returnData"`
	Success    bool        `json:"success"`
	Timestamp  int64       `json:"timestamp"`
}

type ResponseErrorBean struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
}

var Res ResponseSuccessBean

func Success(data interface{}) *ResponseSuccessBean {
	if Res.Message == "" {
		Res.Message = "操作成功!"
	}

	return &ResponseSuccessBean{
		Code:       200,
		Message:    Res.Message,
		Result:     data,
		ReturnData: nil,
		Success:    true,
		Timestamp:  time.Now().Unix(),
	}
}

func Error(errCode uint32, errMsg string) *ResponseErrorBean {
	return &ResponseErrorBean{errCode, errMsg}
}
