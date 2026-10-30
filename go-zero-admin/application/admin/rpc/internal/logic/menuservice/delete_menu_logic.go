package menuservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/model"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteMenuLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteMenuLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteMenuLogic {
	return &DeleteMenuLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteMenuLogic) DeleteMenu(in *adminpb.DeleteMenuRequest) (*adminpb.NoDataResponse, error) {
	if err := l.svcCtx.SysMenuModel.DeleteSubtree(l.ctx, uint64(in.MenuId)); err != nil {
		if errors.Is(err, model.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
		}
		if errors.Is(err, model.ErrMenuHierarchy) {
			return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, err.Error())
		}
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
