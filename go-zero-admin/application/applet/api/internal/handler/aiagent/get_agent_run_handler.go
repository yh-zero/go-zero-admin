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

func GetAgentRunHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.AgentIDRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.ErrorCtx(r.Context(), w, err)
			return
		}

		l := aiagent.NewGetAgentRunLogic(r.Context(), svcCtx)
		resp, err := l.GetAgentRun(&req)
		result.HttpResult(r, w, resp, err)
	}
}
