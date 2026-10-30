package xerr

import "errors"

// CodeError 业务错误：携带错误码和消息
type CodeError struct {
	errCode uint32
	errMsg  string
}

func NewErrCode(errCode uint32) *CodeError {
	return &CodeError{errCode: errCode, errMsg: MapErrMsg(errCode)}
}

func NewErrCodeMsg(errCode uint32, errMsg string) *CodeError {
	return &CodeError{errCode: errCode, errMsg: errMsg}
}

func (e *CodeError) GetErrCode() uint32 { return e.errCode }
func (e *CodeError) GetErrMsg() string  { return e.errMsg }
func (e *CodeError) Error() string      { return e.errMsg }

// AsCodeError 从 error 链中提取 CodeError
func AsCodeError(err error) (*CodeError, bool) {
	var ce *CodeError
	if errors.As(err, &ce) {
		return ce, true
	}
	return nil, false
}
