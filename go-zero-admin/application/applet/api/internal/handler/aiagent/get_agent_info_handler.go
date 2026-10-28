// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package aiagent

import (
	"net/http"

	"go-zero-admin/application/applet/api/internal/logic/aiagent"
	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/pkg/result"
)

// 查询AI开关、模型名称和当前用户可用的只读工具
func GetAgentInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := aiagent.NewGetAgentInfoLogic(r.Context(), svcCtx)
		resp, err := l.GetAgentInfo()
		result.HttpResult(r, w, resp, err)
	}
}
