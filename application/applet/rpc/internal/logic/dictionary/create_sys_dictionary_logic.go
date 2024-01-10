package dictionarylogic

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSysDictionaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSysDictionaryLogic {
	return &CreateSysDictionaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 新建SysDictionary
func (l *CreateSysDictionaryLogic) CreateSysDictionary(in *pb.CreateSysDictionaryRequest) (*pb.NoDataResponse, error) {
	fmt.Println("============ in", in.SysDictionary.Status)
	err := l.svcCtx.DB.First(&model.SysDictionary{}, "type = ?", in.SysDictionary.Type).Error
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errors.New("错误： 存在相同的type")
	}
	var modelSysDictionary model.SysDictionary
	_ = copier.Copy(&modelSysDictionary, in.SysDictionary)
	modelSysDictionary.DeletedAt.Valid = false
	err = l.svcCtx.DB.Create(&modelSysDictionary).Error

	return &pb.NoDataResponse{}, err
}
