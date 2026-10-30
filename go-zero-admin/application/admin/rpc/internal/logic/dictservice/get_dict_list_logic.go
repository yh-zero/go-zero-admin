package dictservicelogic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	where := "deleted_at IS NULL"
	var args []interface{}
	if in.Name != "" {
		where += " AND name LIKE ?"
		args = append(args, "%"+in.Name+"%")
	}
	if in.Code != "" {
		where += " AND code LIKE ?"
		args = append(args, "%"+in.Code+"%")
	}
	if in.Status > 0 {
		where += " AND status=?"
		args = append(args, in.Status)
	}

	var total int64
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM sys_dict WHERE "+where, args...)

	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	type row struct {
		Id          int64
		Name        string
		Code        string
		Items       string
		Description string
		Status      int64
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}
	var rows []row
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &rows,
		"SELECT id, name, code, items, description, status, created_at, updated_at FROM sys_dict WHERE "+where+" ORDER BY id DESC LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
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
			CreatedAt: r.CreatedAt.Unix(), UpdatedAt: r.UpdatedAt.Unix(),
		})
	}
	return &adminpb.GetDictListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
