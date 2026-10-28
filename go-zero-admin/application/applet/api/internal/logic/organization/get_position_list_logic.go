// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organization

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPositionListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询岗位列表
func NewGetPositionListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPositionListLogic {
	return &GetPositionListLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPositionListLogic) GetPositionList(req *types.OrganizationListRequest) (resp *types.PositionListResponse, err error) {
	result, err := l.svcCtx.AppletOrganizationRPC.GetPositionList(l.ctx, &pb.OrganizationListRequest{Keyword: req.Keyword, Status: req.Status})
	if err != nil {
		return nil, err
	}
	out := &types.PositionListResponse{List: []types.Position{}}
	for _, p := range result.List {
		if value := positionOutput(p); value != nil {
			out.List = append(out.List, *value)
		}
	}
	return out, nil
}
