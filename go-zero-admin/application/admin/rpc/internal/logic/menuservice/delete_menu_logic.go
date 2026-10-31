package menuservicelogic

import (
	"context"
	"errors"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
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
	// 事务软删子树 + 清角色关联 + 清缓存（见 model.SoftDeleteSubtree）
	if _, err := l.svcCtx.SysMenuModel.SoftDeleteSubtree(l.ctx, uint64(in.MenuId)); err != nil {
		if errors.Is(err, sqlx.ErrNotFound) {
			return nil, xerr.NewErrCode(xerr.MENU_NOT_FOUND)
		}
		l.Errorf("delete menu subtree error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
