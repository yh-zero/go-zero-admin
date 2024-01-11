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

type UpdateSysDictionaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryLogic {
	return &UpdateSysDictionaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新SysDictionary
func (l *UpdateSysDictionaryLogic) UpdateSysDictionary(in *pb.UpdateSysDictionaryRequest) (*pb.NoDataResponse, error) {
	var modelSysDictionary model.SysDictionary
	sysDictionaryMap := map[string]interface{}{
		"Name":   in.SysDictionary.Name,
		"Type":   in.SysDictionary.Type,
		"Status": in.SysDictionary.Status,
		"Desc":   in.SysDictionary.Desc,
	}
	db := l.svcCtx.DB.Where("id = ?", in.SysDictionary.ID).First(&modelSysDictionary)
	if modelSysDictionary.Type != in.SysDictionary.Type {
		if !errors.Is(l.svcCtx.DB.First(&model.SysDictionary{}, "type = ?", in.SysDictionary.Type).Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("错误： 存在相同的type，不允许修改")
		}
	}
	err := db.Updates(sysDictionaryMap).Error

	return &pb.NoDataResponse{}, err
}
