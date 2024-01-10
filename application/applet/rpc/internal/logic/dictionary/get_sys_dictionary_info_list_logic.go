package dictionarylogic

import (
	"context"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetSysDictionaryInfoListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSysDictionaryInfoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryInfoListLogic {
	return &GetSysDictionaryInfoListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取SysDictionaryInfo列表
func (l *GetSysDictionaryInfoListLogic) GetSysDictionaryInfoList(in *pb.GetSysDictionaryInfoListRequest) (*pb.GetSysDictionaryInfoListResponse, error) {
	offset := int(in.PageRequest.PageSize * (in.PageRequest.PageNo - 1))
	var total int64

	// 创建db
	db := l.svcCtx.DB.Model(&model.SysDictionaryInfo{})
	var modelSysDictionaryInfoList []model.SysDictionaryInfo
	// 如果有条件搜索 下方会自动创建搜索语句
	if in.SysDictionaryInfo.Label != "" {
		db = db.Where("label LIKE ?", "%"+in.SysDictionaryInfo.Label+"%")
	}
	if in.SysDictionaryInfo.Value != 0 {
		db = db.Where("value = ?", in.SysDictionaryInfo.Value)
	}
	if in.SysDictionaryInfo.Status != 0 {
		db = db.Where("status = ?", in.SysDictionaryInfo.Status)
	}
	if in.SysDictionaryInfo.SysDictionaryID != 0 {
		db = db.Where("sys_dictionary_id = ?", in.SysDictionaryInfo.SysDictionaryID)
	}
	err := db.Count(&total).Error
	if err != nil {
		return nil, err
	}
	err = db.Limit(int(in.PageRequest.PageSize)).Offset(offset).Order("sort").Find(&modelSysDictionaryInfoList).Error

	var pbSysDictionaryInfoList []*pb.SysDictionaryInfo
	_ = copier.Copy(&pbSysDictionaryInfoList, modelSysDictionaryInfoList)

	return &pb.GetSysDictionaryInfoListResponse{
		SysDictionaryInfoList: pbSysDictionaryInfoList,
		Total:                 total,
	}, nil
}
