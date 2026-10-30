// grpc.go 业务错误码经 gRPC status details 结构化传递（避免退化为 Unknown）
package xerr

import (
	"strconv"

	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// 错误详情在 details 中的 key
const (
	detailDomain  = "go-zero-admin"
	detailReason  = "BUSINESS_ERROR"
	detailCodeKey = "code"
	detailMsgKey  = "msg"
)

// GRPCStatus 实现 gRPC status 接口，使 CodeError 能携带业务码跨服务传递
func (e *CodeError) GRPCStatus() *status.Status {
	s, _ := status.New(codes.Unknown, e.errMsg).WithDetails(&errdetails.ErrorInfo{
		Reason: detailReason,
		Domain: detailDomain,
		Metadata: map[string]string{
			detailCodeKey: strconv.FormatUint(uint64(e.errCode), 10),
			detailMsgKey:  e.errMsg,
		},
	})
	return s
}

// FromGRPCError 从 gRPC error 中还原业务错误码（API 侧调用）
func FromGRPCError(err error) (uint32, string, bool) {
	if err == nil {
		return 0, "", false
	}
	st, ok := status.FromError(err)
	if !ok {
		return 0, "", false
	}
	for _, d := range st.Details() {
		det, ok := d.(*errdetails.ErrorInfo)
		if !ok || det.Reason != detailReason {
			continue
		}
		code, _ := strconv.ParseUint(det.Metadata[detailCodeKey], 10, 32)
		msg := det.Metadata[detailMsgKey]
		if msg == "" {
			msg = st.Message()
		}
		return uint32(code), msg, true
	}
	return 0, st.Message(), false
}

// Wrap 将业务错误包成带 details 的 gRPC status error（RPC 边界调用）
func Wrap(err error) error {
	if err == nil {
		return nil
	}
	if ce, ok := AsCodeError(err); ok {
		return ce.GRPCStatus().Err()
	}
	return err
}
