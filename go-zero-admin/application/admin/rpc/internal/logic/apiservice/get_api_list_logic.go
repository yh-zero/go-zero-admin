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
	where := "1=1"
	var args []interface{}
	if in.Group != "" {
		where += " AND group_name=?"
		args = append(args, in.Group)
	}
	if in.Method != "" {
		where += " AND method=?"
		args = append(args, in.Method)
	}
	if in.Path != "" {
		where += " AND path LIKE ?"
		args = append(args, "%"+in.Path+"%")
	}
	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM sys_api WHERE "+where, args...)
	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	type apiRow struct {
		Id        int64  `db:"id"`
		GroupName string `db:"group_name"`
		Method    string `db:"method"`
		Path      string `db:"path"`
		Title     string `db:"title"`
		CreatedAt int64  `db:"created_at"`
	}
	var apiRows []apiRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &apiRows,
		"SELECT id, group_name, method, path, title, created_at FROM sys_api WHERE "+where+" ORDER BY id LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.ApiInfo, 0, len(apiRows))
	for _, r := range apiRows {
		list = append(list, &adminpb.ApiInfo{Id: r.Id, Group: r.GroupName, Method: r.Method, Path: r.Path, Title: r.Title, CreatedAt: r.CreatedAt})
	}
	return &adminpb.GetApiListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
