package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetMyMenusLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyMenusLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyMenusLogic {
	return &GetMyMenusLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMyMenusLogic) GetMyMenus(in *adminpb.GetMyMenusRequest) (*adminpb.GetMyMenusResponse, error) {
	rows, err := l.svcCtx.SysMenuModel.FindByUser(l.ctx, in.GetActor().GetUserId())
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMyMenusResponse{List: menuInfoList(rows)}, nil
}
