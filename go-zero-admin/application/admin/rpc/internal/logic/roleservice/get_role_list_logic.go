package roleservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetRoleListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetRoleListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetRoleListLogic {
	return &GetRoleListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetRoleListLogic) GetRoleList(in *adminpb.GetRoleListRequest) (*adminpb.GetRoleListResponse, error) {
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
	_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &total, "SELECT COUNT(*) FROM sys_role WHERE "+where, args...)
	page := in.Page.Page
	pageSize := in.Page.PageSize
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	type roleRow struct {
		Id          int64  `db:"id"`
		Name        string `db:"name"`
		Code        string `db:"code"`
		Description string `db:"description"`
		DataScope   string `db:"data_scope"`
		Sort        int32  `db:"sort"`
		Status      int32  `db:"status"`
		CreatedAt   int64  `db:"created_at"`
		UpdatedAt   int64  `db:"updated_at"`
	}
	var roleRows []roleRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &roleRows,
		"SELECT id, name, code, description, data_scope, sort, status, UNIX_TIMESTAMP(created_at), UNIX_TIMESTAMP(updated_at) FROM sys_role WHERE "+where+" ORDER BY sort, id LIMIT ? OFFSET ?",
		append(args, pageSize, offset)...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.RoleInfo, 0, len(roleRows))
	for _, r := range roleRows {
		list = append(list, &adminpb.RoleInfo{Id: r.Id, Name: r.Name, Code: r.Code, Description: r.Description, DataScope: r.DataScope, Sort: r.Sort, Status: r.Status, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt})
	}
	return &adminpb.GetRoleListResponse{List: list, Page: &adminpb.PageResponse{Total: total}}, nil
}
