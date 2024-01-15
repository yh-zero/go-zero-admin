package dictionarylogic

import (
	"context"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSysDictionaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSysDictionaryLogic {
	return &DeleteSysDictionaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新SysDictionary
func (l *DeleteSysDictionaryLogic) DeleteSysDictionary(in *pb.DeleteSysDictionaryRequest) (*pb.NoDataResponse, error) {
	var modelSysDictionary model.SysDictionary
	err := l.svcCtx.DB.Where("id = ?", in.ID).Preload("SysDictionaryInfoList").First(&modelSysDictionary).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("无效id")
	}
	if err != nil {
		return nil, err
	}
	err = l.svcCtx.DB.Delete(&modelSysDictionary).Error
	if err != nil {
		return nil, err
	}
	if modelSysDictionary.SysDictionaryInfoList != nil {
		err = l.svcCtx.DB.Where("sys_dictionary_id=?", modelSysDictionary.ID).Delete(modelSysDictionary.SysDictionaryInfoList).Error
	}

	return &pb.NoDataResponse{}, err
}
