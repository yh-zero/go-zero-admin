package ai

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/ai/rpc/aipb"
)

type ListToolsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewListToolsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListToolsLogic {
	return &ListToolsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}
func (l *ListToolsLogic) ListTools(req *types.AiListToolsReq) (*types.AiListToolsResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	actor, err := actorForAi(l.ctx, l.svcCtx)
	if err != nil {
		return nil, err
	}
	out, err := l.svcCtx.AiRpc.ListTools(l.ctx, &aipb.ListToolsRequest{Actor: actor})
	if err != nil {
		return nil, err
	}
	list := make([]*types.AiToolInfo, 0, len(out.List))
	for _, item := range out.List {
		list = append(list, &types.AiToolInfo{Name: item.Name, Description: item.Description, ParametersSchema: item.ParametersSchema})
	}
	return &types.AiListToolsResp{List: list, Capabilities: capabilitiesInfo(out.Capabilities)}, nil
}
