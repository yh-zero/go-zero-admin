package dept

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDeptLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateDeptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDeptLogic {
	return &UpdateDeptLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateDeptLogic) UpdateDept(req *types.UpdateDeptReq) error {
	_, err := l.svcCtx.DeptRpc.UpdateDept(l.ctx, &adminpb.UpdateDeptRequest{
		DeptId: req.Id, ParentId: req.ParentId, Name: req.Name, Leader: req.Leader,
		Phone: req.Phone, Email: req.Email, Sort: req.Sort, Status: req.Status,
	})
	return err
}
