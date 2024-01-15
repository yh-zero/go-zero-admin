package menu

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetBaseMenuTreeLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetBaseMenuTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBaseMenuTreeLogic {
	return &GetBaseMenuTreeLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetBaseMenuTreeLogic) GetBaseMenuTree(req *types.GetBaseMenuTreeRequest) (resp *types.GetBaseMenuTreeResponse, err error) {
	baseMenuTree, err := l.svcCtx.AppletMenuRPC.GetBaseMenuTree(l.ctx, &pb.NoDataResponse{})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletMenuRPC.GetBaseMenuTree err: %v", err)
		return nil, err
	}

	var sysBaseMenuList []types.SysBaseMenu
	_ = copier.Copy(&sysBaseMenuList, baseMenuTree.SysBaseMenuList)

	return &types.GetBaseMenuTreeResponse{
		SysBaseMenuList: sysBaseMenuList,
	}, err
}
