// Code scaffolded by goctl. Safe to edit.
// goctl 1.10.2

package dept

import (
	"context"
	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetMyDeptsLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetMyDeptsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyDeptsLogic {
	return &GetMyDeptsLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetMyDeptsLogic) GetMyDepts(req *types.GetMyDeptsReq) (resp *types.GetMyDeptsResp, err error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	result, err := l.svcCtx.DeptRpc.GetMyDepts(l.ctx, &adminpb.GetMyDeptsRequest{Actor: middleware.ActorFromContext(l.ctx)})
	if err != nil {
		return nil, err
	}
	list := make([]*types.DeptInfo, 0, len(result.List))
	for _, dept := range result.List {
		list = append(list, mapDeptInfo(dept))
	}
	return &types.GetMyDeptsResp{List: list}, nil
}
