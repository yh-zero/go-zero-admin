package xerr

import (
	"fmt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/**
常用通用固定错误
*/

type CodeError struct {
	errCode uint32
	errMsg  string
}

// 返回给前端的错误码
func (e *CodeError) GetErrCode() uint32 {
	return e.errCode
}

// 返回给前端显示端错误信息
func (e *CodeError) GetErrMsg() string {
	return e.errMsg
}

func (e *CodeError) Error() string {
	return fmt.Sprintf("ErrCode:%d，ErrMsg:%s", e.errCode, e.errMsg)
}

// GRPCStatus 保留业务错误码，API 只向客户端公开这些经过定义的错误。
func (e *CodeError) GRPCStatus() *status.Status {
	return status.New(codes.Code(e.errCode), e.errMsg)
}

func NewErrCode(errCode uint32) *CodeError {
	return &CodeError{errCode: errCode, errMsg: MapErrMsg(errCode)}
}

// 暂时屏蔽 都用一个 errCode  在errCode、errMsg 定义就好
func NewErrCodeMsg(errCode uint32, errMsg string) *CodeError {
	return &CodeError{errCode: errCode, errMsg: errMsg}
}
func NewErrMsg(errMsg string) *CodeError {
	return &CodeError{errCode: SERVER_COMMON_ERROR, errMsg: errMsg}
}
