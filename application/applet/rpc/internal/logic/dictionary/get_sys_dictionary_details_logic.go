package dictionarylogic

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryDetailsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSysDictionaryDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryDetailsLogic {
	return &GetSysDictionaryDetailsLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 根据ID或者type获取SysDictionary
func (l *GetSysDictionaryDetailsLogic) GetSysDictionaryDetails(in *pb.GetSysDictionaryDetailsRequest) (*pb.GetSysDictionaryDetailsResponse, error) {
	var modelSysDictionary model.SysDictionary
	err := l.svcCtx.DB.Where("(type = ? OR id = ?) and status = ?", in.Type, in.ID, in.Status).Preload("SysDictionaryInfoList", func(db *gorm.DB) *gorm.DB {
		return db.Where("status = ?", 1).Order("sort")
	}).First(&modelSysDictionary).Error
	if err != nil {
		return nil, err
	}
	var pbSysDictionary pb.SysDictionary
	_ = copier.Copy(&pbSysDictionary, modelSysDictionary)

	return &pb.GetSysDictionaryDetailsResponse{SysDictionary: &pbSysDictionary}, err
}
