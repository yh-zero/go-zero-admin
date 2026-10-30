package dict

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateDictLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateDictLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateDictLogic {
	return &CreateDictLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateDictLogic) CreateDict(req *types.CreateDictReq) error {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	actor := &adminpb.Actor{UserId: userID}
	_, err := l.svcCtx.DictRpc.CreateDict(l.ctx, &adminpb.CreateDictRequest{
		Actor: actor, Name: req.Name, Code: req.Code, Items: mapDictItems(req.Items),
		Description: req.Description, Status: req.Status,
	})
	return err
}
