package dictservicelogic

import (
	"context"
	"encoding/json"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetDictListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDictListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDictListLogic {
	return &GetDictListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetDictListLogic) GetDictList(in *adminpb.GetDictListRequest) (*adminpb.GetDictListResponse, error) {
	rows, total, err := l.svcCtx.SysDictModel.ListDictPage(l.ctx, model.DictListFilter{
		Name: in.Name, Code: in.Code, Status: int64(in.Status),
	}, in.GetPage().GetPage(), in.GetPage().GetPageSize())
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}

	list := make([]*adminpb.DictInfo, 0, len(rows))
	for _, r := range rows {
		var items []*adminpb.DictItem
		_ = json.Unmarshal([]byte(r.Items), &items)
		list = append(list, &adminpb.DictInfo{
			Id: r.Id, Name: r.Name, Code: r.Code, Items: items,
			Description: r.Description, Status: int32(r.Status),
			CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		})
	}
	return &adminpb.GetDictListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
