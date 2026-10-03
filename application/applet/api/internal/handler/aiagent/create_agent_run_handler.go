// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiagent

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/aiagent"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/pkg/result"

	"github.com/zeromicro/go-zero/rest/httpx"
)

// 提交有界异步Agent任务，同一requestId幂等
func CreateAgentRunHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateAgentRunRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := aiagent.NewCreateAgentRunLogic(r.Context(), svcCtx)
		resp, err := l.CreateAgentRun(&req)
		result.HttpResult(r, w, resp, err)
	}
}
