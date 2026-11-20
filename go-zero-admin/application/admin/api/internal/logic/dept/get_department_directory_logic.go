// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package dept

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetDepartmentDirectoryLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetDepartmentDirectoryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetDepartmentDirectoryLogic {
	return &GetDepartmentDirectoryLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetDepartmentDirectoryLogic) GetDepartmentDirectory(req *types.GetDepartmentDirectoryReq) (*types.GetDepartmentDirectoryResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	result, err := l.svcCtx.DeptRpc.GetDepartmentDirectory(l.ctx, &adminpb.GetDepartmentDirectoryRequest{
		Actor: middleware.ActorFromContext(l.ctx), Keyword: req.Keyword, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		return nil, err
	}
	list := make([]*types.DepartmentDirectoryMember, 0, len(result.List))
	for _, member := range result.List {
		list = append(list, &types.DepartmentDirectoryMember{Id: member.Id, Name: member.Name, IsLeader: member.IsLeader})
	}
	return &types.GetDepartmentDirectoryResp{DeptName: result.DeptName, List: list, Total: result.Total}, nil
}
