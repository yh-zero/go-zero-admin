// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package permission

import (
	"context"

	"go-zero-admin/application/applet/rpc/pb"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetPermissionHistoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetPermissionHistoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetPermissionHistoryLogic {
	return &GetPermissionHistoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetPermissionHistoryLogic) GetPermissionHistory(req *types.GetPermissionHistoryRequest) (resp *types.GetPermissionHistoryResponse, err error) {
	result, err := l.svcCtx.AppletPermissionRPC.GetPermissionHistory(l.ctx, &pb.GetPermissionHistoryRequest{AuthorityId: req.AuthorityId, Kind: req.Kind, PageRequest: &pb.PageRequest{PageNo: req.PageNo, PageSize: req.PageSize}})
	if err != nil {
		return nil, err
	}
	resp = &types.GetPermissionHistoryResponse{List: []types.PermissionChange{}, Total: result.Total}
	for _, row := range result.List {
		resp.List = append(resp.List, historyChange(row))
	}
	return resp, nil
}
