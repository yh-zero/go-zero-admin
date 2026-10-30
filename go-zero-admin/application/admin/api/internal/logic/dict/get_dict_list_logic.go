package dict

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDictListLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDictListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDictListLogic {
	return &GetDictListLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetDictListLogic) GetDictList(req *types.GetDictListReq) (resp *types.GetDictListResp, err error) {
	rpcResp, err := l.svcCtx.DictRpc.GetDictList(l.ctx, &adminpb.GetDictListRequest{
		Page: &adminpb.PageRequest{Page: req.Page, PageSize: req.PageSize},
		Name: req.Name, Code: req.Code, Status: req.Status,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.DictInfo, 0, len(rpcResp.List))
	for _, d := range rpcResp.List {
		list = append(list, mapDictInfo(d))
	}
	return &types.GetDictListResp{List: list, Total: rpcResp.Page.Total}, nil
}

func mapDictItem(d *adminpb.DictItem) *types.DictItem {
	return &types.DictItem{Label: d.Label, Value: d.Value, Sort: d.Sort}
}
func mapDictInfo(d *adminpb.DictInfo) *types.DictInfo {
	items := make([]*types.DictItem, 0, len(d.Items))
	for _, it := range d.Items {
		items = append(items, mapDictItem(it))
	}
	return &types.DictInfo{Id: d.Id, Name: d.Name, Code: d.Code, Items: items, Description: d.Description, Status: d.Status, CreatedAt: d.CreatedAt, UpdatedAt: d.UpdatedAt}
}
func mapDictItems(items []*types.DictItem) []*adminpb.DictItem {
	out := make([]*adminpb.DictItem, 0, len(items))
	for _, it := range items {
		out = append(out, &adminpb.DictItem{Label: it.Label, Value: it.Value, Sort: it.Sort})
	}
	return out
}
