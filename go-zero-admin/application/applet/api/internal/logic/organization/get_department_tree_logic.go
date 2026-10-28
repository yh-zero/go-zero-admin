// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package organization

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDepartmentTreeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 查询部门树
func NewGetDepartmentTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDepartmentTreeLogic {
	return &GetDepartmentTreeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDepartmentTreeLogic) GetDepartmentTree(req *types.OrganizationListRequest) (resp *types.DepartmentListResponse, err error) {
	result, err := l.svcCtx.AppletOrganizationRPC.GetDepartmentTree(l.ctx, &pb.OrganizationListRequest{Keyword: req.Keyword, Status: req.Status})
	if err != nil {
		return nil, err
	}
	out := &types.DepartmentListResponse{List: []types.Department{}}
	for _, d := range result.List {
		if value := departmentOutput(d); value != nil {
			out.List = append(out.List, *value)
		}
	}
	return out, nil
}
