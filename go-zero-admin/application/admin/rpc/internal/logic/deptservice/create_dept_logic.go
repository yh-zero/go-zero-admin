package deptservicelogic

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type CreateDeptLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateDeptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDeptLogic {
	return &CreateDeptLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateDeptLogic) CreateDept(in *adminpb.CreateDeptRequest) (*adminpb.NoDataResponse, error) {
	// 先插记录拿 ID
	result, err := l.svcCtx.DB.ExecCtx(l.ctx,
		"INSERT INTO sys_dept (parent_id, name, leader, phone, email, sort, status, path) VALUES (?,?,?,?,?,?,?,?)",
		in.ParentId, in.Name, in.Leader, in.Phone, in.Email, in.Sort, in.Status, "")
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	id, _ := result.LastInsertId()
	// 算 path
	var path string
	if in.ParentId == 0 {
		path = fmt.Sprintf("/%d/", id)
	} else {
		var parentPath string
		_ = l.svcCtx.DB.QueryRowCtx(l.ctx, &parentPath, "SELECT path FROM sys_dept WHERE id=?", in.ParentId)
		path = fmt.Sprintf("%s%d/", parentPath, id)
	}
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_dept SET path=? WHERE id=?", path, id)
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
