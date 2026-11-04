package dict

import (
	"context"
	"go-zero-admin/pkg/xerr"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	"go-zero-admin/application/admin/rpc/adminpb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateDictLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateDictLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateDictLogic {
	return &UpdateDictLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *UpdateDictLogic) UpdateDict(req *types.UpdateDictReq) (*types.EmptyResp, error) {
	if err := l.svcCtx.Validator.StructCtx(l.ctx, req); err != nil {
		return nil, xerr.NewErrCode(xerr.REQUEST_PARAM_ERROR)
	}
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	actor := &adminpb.Actor{UserId: userID}
	_, err := l.svcCtx.DictRpc.UpdateDict(l.ctx, &adminpb.UpdateDictRequest{
		Actor: actor, DictId: req.Id, Name: req.Name, Code: req.Code, Items: mapDictItems(req.Items), UpdateItems: req.Items != nil,
		Description: req.Description, Status: req.Status,
	})
	return &types.EmptyResp{}, err
}
