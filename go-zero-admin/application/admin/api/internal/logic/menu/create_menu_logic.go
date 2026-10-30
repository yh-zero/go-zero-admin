package menu

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateMenuLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateMenuLogic {
	return &CreateMenuLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *CreateMenuLogic) CreateMenu(req *types.CreateMenuReq) error {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	actor := &adminpb.Actor{UserId: userID}
	_, err := l.svcCtx.MenuRpc.CreateMenu(l.ctx, &adminpb.CreateMenuRequest{
		Actor: actor, ParentId: req.ParentId, Title: req.Title, Name: req.Name, Type: req.Type,
		Path: req.Path, Component: req.Component, PermCode: req.PermCode, Icon: req.Icon,
		Sort: req.Sort, Visible: req.Visible, Status: req.Status,
	})
	return err
}
