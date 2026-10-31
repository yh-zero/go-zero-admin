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
	rows, err := l.svcCtx.SysDeptModel.ListDeptTree(l.ctx, int64(in.Status))
	if err != nil {
		l.Errorf("dept tree error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	list := make([]*adminpb.DeptInfo, 0, len(rows))
	for _, r := range rows {
		list = append(list, &adminpb.DeptInfo{
			Id: r.Id, ParentId: r.ParentId, Name: r.Name, Leader: r.Leader,
			Phone: r.Phone, Email: r.Email, Sort: r.Sort, Status: r.Status,
			Path: r.Path, CreatedAt: r.CreatedAt,
		})
	}
	return &adminpb.GetDeptTreeResponse{List: list}, nil
}
