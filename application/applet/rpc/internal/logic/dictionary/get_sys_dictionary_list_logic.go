package dictionarylogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSysDictionaryListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryListLogic {
	return &GetSysDictionaryListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取SysDictionary列表 - 全部
func (l *GetSysDictionaryListLogic) GetSysDictionaryList(in *pb.NoDataResponse) (*pb.DictionaryListResponse, error) {
	var modelSysDictionaryList []model.SysDictionary
	err := l.svcCtx.DB.Find(&modelSysDictionaryList).Error
	var sysDictionaryList []*pb.SysDictionary
	_ = copier.Copy(&sysDictionaryList, modelSysDictionaryList)

	return &pb.DictionaryListResponse{
		SysDictionaryList: sysDictionaryList,
	}, err
}
