package apiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetApiListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetApiListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApiListLogic {
	return &GetApiListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetApiListLogic) GetApiList(in *adminpb.GetApiListRequest) (*adminpb.GetApiListResponse, error) {
	page := in.GetPage()
	rows, total, err := l.svcCtx.SysApiModel.FindList(l.ctx, in.Group, in.Method, in.Path, page.GetPage(), page.GetPageSize())
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.ApiInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &adminpb.ApiInfo{Id: r.Id, Group: r.GroupName, Method: r.Method, Path: r.Path, Title: r.Title, CreatedAt: r.CreatedAt})
	}
	return &adminpb.GetApiListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
