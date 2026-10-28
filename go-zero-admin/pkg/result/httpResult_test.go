package result

import (
	"encoding/json"
	"errors"
	"go-zero-admin/pkg/result/xerr"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http/httptest"
	"testing"
)

func TestBusinessErrorsSurviveRPCWithoutLeakingInternalErrors(t *testing.T) {
	cases := []struct {
		name    string
		err     error
		code    uint32
		message string
	}{
		{"rpc business", status.Convert(xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "角色不存在")).Err(), xerr.REUQEST_PARAM_ERROR, "角色不存在"},
		{"internal rpc", status.Error(codes.Unknown, "mysql credentials and query"), xerr.SERVER_COMMON_ERROR, xerr.MapErrMsg(xerr.SERVER_COMMON_ERROR)},
		{"internal local", errors.New("internal storage credentials"), xerr.SERVER_COMMON_ERROR, xerr.MapErrMsg(xerr.SERVER_COMMON_ERROR)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			HttpResult(httptest.NewRequest("GET", "/", nil), response, nil, c.err)
			var output ResponseErrorBean
			if err := json.Unmarshal(response.Body.Bytes(), &output); err != nil {
				t.Fatal(err)
			}
			if output.Code != c.code || output.Message != c.message {
				t.Fatalf("unexpected response: %+v", output)
			}
		})
	}
}
