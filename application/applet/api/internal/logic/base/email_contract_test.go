package base

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/zeromicro/go-zero/rest/httpx"
	"go-zero-admin/application/applet/api/internal/types"
)

func TestEmailCodeContractExcludesClientIPFromJSON(t *testing.T) {
	r := httptest.NewRequest("POST", "/v1/base/sendEmailCode", strings.NewReader(`{"email":"member@example.test","scene":"register","isForce":true,"clientIP":"198.51.100.11","ClientIP":"198.51.100.12"}`))
	r.Header.Set("Content-Type", "application/json")
	var req types.SendEmailCodeRequest
	if err := httpx.Parse(r, &req); err != nil {
		t.Fatal(err)
	}
	if req.Email != "member@example.test" || req.Scene != "register" || !req.IsForce || req.ClientIP != "" {
		t.Fatalf("email contract lost fields or accepted caller-controlled rate-limit IP: %+v", req)
	}
}
