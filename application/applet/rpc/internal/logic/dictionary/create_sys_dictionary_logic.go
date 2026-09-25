package dictionarylogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type CreateSysDictionaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSysDictionaryLogic {
	return &CreateSysDictionaryLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateSysDictionaryLogic) CreateSysDictionary(in *pb.CreateSysDictionaryRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if in.SysDictionary != nil {
			in.SysDictionary.ID = 0
			if in.SysDictionary.Status == 0 {
				in.SysDictionary.Status = 1
			}
		}
		if err := validateDictionary(tx, in.SysDictionary); err != nil {
			return err
		}
		value := in.SysDictionary
		return tx.Create(&model.SysDictionary{Name: value.Name, Type: value.Type, Status: value.Status, Desc: value.Desc}).Error
	})
	return &pb.NoDataResponse{}, err
}
