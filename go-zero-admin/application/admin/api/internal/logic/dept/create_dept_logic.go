package dept

import (
	"context"

	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	adminpb "go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDeptLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateDeptLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDeptLogic {
	return &CreateDeptLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateDeptLogic) CreateDept(req *types.CreateDeptReq) error {
	_, err := l.svcCtx.DeptRpc.CreateDept(l.ctx, &adminpb.CreateDeptRequest{
		ParentId: req.ParentId, Name: req.Name, Leader: req.Leader,
		Phone: req.Phone, Email: req.Email, Sort: req.Sort, Status: req.Status,
	})
	return err
}
