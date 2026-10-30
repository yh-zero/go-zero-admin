package deptservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetDeptTreeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetDeptTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDeptTreeLogic {
	return &GetDeptTreeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetDeptTreeLogic) GetDeptTree(in *adminpb.GetDeptTreeRequest) (*adminpb.GetDeptTreeResponse, error) {
	where := "deleted_at IS NULL"
	var args []interface{}
	if in.Status > 0 {
		where += " AND status=?"
		args = append(args, in.Status)
	}
	type deptRow struct {
		Id        int64  `db:"id"`
		ParentId  int64  `db:"parent_id"`
		Name      string `db:"name"`
		Leader    string `db:"leader"`
		Phone     string `db:"phone"`
		Email     string `db:"email"`
		Sort      int32  `db:"sort"`
		Status    int32  `db:"status"`
		Path      string `db:"path"`
		CreatedAt int64  `db:"created_at"`
	}
	var deptRows []deptRow
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &deptRows,
		"SELECT id, parent_id, name, leader, phone, email, sort, status, path, UNIX_TIMESTAMP(created_at) FROM sys_dept WHERE "+where+" ORDER BY sort, id", args...)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.DeptInfo, 0, len(deptRows))
	for _, r := range deptRows {
		list = append(list, &adminpb.DeptInfo{Id: r.Id, ParentId: r.ParentId, Name: r.Name, Leader: r.Leader, Phone: r.Phone, Email: r.Email, Sort: r.Sort, Status: r.Status, Path: r.Path, CreatedAt: r.CreatedAt})
	}
	return &adminpb.GetDeptTreeResponse{List: list}, nil
}
