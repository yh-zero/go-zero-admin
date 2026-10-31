package aiservicelogic

import (
	"context"
	"encoding/json"
	"github.com/cloudwego/eino/schema"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/ai/rpc/aipb"
	"go-zero-admin/application/ai/rpc/internal/svc"
	"go-zero-admin/pkg/agenttools"
	"go-zero-admin/pkg/xerr"
	"sort"
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
	if err := requireAiActor(in.Actor); err != nil {
		return nil, err
	}
	list := make([]*aipb.ToolInfo, 0)
	for _, tool := range agenttools.List() {
		parameters, err := schema.NewParamsOneOfByParams(tool.Params()).ToJSONSchema()
		if err != nil {
			return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		encoded, err := json.Marshal(parameters)
		if err != nil {
			return nil, xerr.NewErrCode(xerr.SERVER_COMMON_ERROR)
		}
		list = append(list, &aipb.ToolInfo{Name: tool.Name(), Description: tool.Description(), ParametersSchema: string(encoded)})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Name < list[j].Name })
	return &aipb.ListToolsResponse{List: list}, nil
}
