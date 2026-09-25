package menulogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type GetBaseMenuTreeLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetBaseMenuTreeLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetBaseMenuTreeLogic {
	return &GetBaseMenuTreeLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetBaseMenuTreeLogic) GetBaseMenuTree(in *pb.NoDataResponse) (*pb.GetBaseMenuTreeResponse, error) {
	var all []model.SysBaseMenu
	if err := l.svcCtx.DB.Order("sort,id").Preload("MenuBtn").Preload("Parameters").Find(&all).Error; err != nil {
		return nil, err
	}
	tree := baseMenuTree(all, 0)
	result := &pb.GetBaseMenuTreeResponse{}
	if err := copier.Copy(&result.SysBaseMenuList, tree); err != nil {
		return nil, err
	}
	return result, nil
}
