package dictionarylogic

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryInfoListDetailsByIdLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSysDictionaryInfoListDetailsByIdLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryInfoListDetailsByIdLogic {
	return &GetSysDictionaryInfoListDetailsByIdLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据id获取SysDictionaryInfo详情
func (l *GetSysDictionaryInfoListDetailsByIdLogic) GetSysDictionaryInfoListDetailsById(in *pb.GetSysDictionaryInfoListDetailsByIdRequest) (*pb.GetSysDictionaryInfoListDetailsByIdResponse, error) {
	var modelSysDictionaryInfo model.SysDictionaryInfo
	err := l.svcCtx.DB.Where("id = ?", in.ID).First(&modelSysDictionaryInfo).Error
	var pbSysDictionaryInfo pb.SysDictionaryInfo
	_ = copier.Copy(&pbSysDictionaryInfo, modelSysDictionaryInfo)

	return &pb.GetSysDictionaryInfoListDetailsByIdResponse{
		SysDictionaryInfo: &pbSysDictionaryInfo,
	}, err
}