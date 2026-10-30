package dept

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDeptTreeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDeptTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeptTreeLogic {
	return &GetDeptTreeLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetDeptTreeLogic) GetDeptTree(req *types.GetDeptTreeReq) (resp *types.GetDeptTreeResp, err error) {
	rpcResp, err := l.svcCtx.DeptRpc.GetDeptTree(l.ctx, &adminpb.GetDeptTreeRequest{Status: req.Status})
	if err != nil {
		return nil, err
	}
	list := make([]*types.DeptInfo, 0, len(rpcResp.List))
	for _, d := range rpcResp.List {
		list = append(list, mapDeptInfo(d))
	}
	return &types.GetDeptTreeResp{List: list}, nil
}

func mapDeptInfo(d *adminpb.DeptInfo) *types.DeptInfo {
	return &types.DeptInfo{
		Id: d.Id, ParentId: d.ParentId, Name: d.Name, Leader: d.Leader,
		Phone: d.Phone, Email: d.Email, Sort: d.Sort, Status: d.Status,
		Path: d.Path, CreatedAt: d.CreatedAt,
	}
}
