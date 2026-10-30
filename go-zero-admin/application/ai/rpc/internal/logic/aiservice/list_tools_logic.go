package aiservicelogic

import (
	"context"
	"go-zero-admin/pkg/agenttools"

	"github.com/zeromicro/go-zero/core/logx"

	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
)

type ListToolsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListToolsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListToolsLogic {
	return &ListToolsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *ListToolsLogic) ListTools(in *aipb.ListToolsRequest) (*aipb.ListToolsResponse, error) {
	list := make([]*aipb.ToolInfo, 0)
	for _, t := range agenttools.List() {
		list = append(list, &aipb.ToolInfo{
			Name: t.Name(), Description: t.Description(), ParametersSchema: "",
		})
	}
	return &aipb.ListToolsResponse{List: list}, nil
}
