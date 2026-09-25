package menulogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type GetMenuBaseInfoListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMenuBaseInfoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMenuBaseInfoListLogic {
	return &GetMenuBaseInfoListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMenuBaseInfoListLogic) GetMenuBaseInfoList(in *pb.NoDataResponse) (*pb.GetMenuBaseInfoListResponse, error) {
	var all []model.SysBaseMenu
	if err := l.svcCtx.DB.Order("sort,id").Preload("MenuBtn").Preload("Parameters").Find(&all).Error; err != nil {
		return nil, err
	}
	tree := baseMenuTree(all, 0)
	result := &pb.GetMenuBaseInfoListResponse{Total: int64(len(all))}
	if err := copier.Copy(&result.SysBaseMenu, tree); err != nil {
		return nil, err
	}
	return result, nil
}
