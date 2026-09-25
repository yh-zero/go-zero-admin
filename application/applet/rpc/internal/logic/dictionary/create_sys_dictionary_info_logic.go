package dictionarylogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type CreateSysDictionaryInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSysDictionaryInfoLogic {
	return &CreateSysDictionaryInfoLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateSysDictionaryInfoLogic) CreateSysDictionaryInfo(in *pb.CreateSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		if in.SysDictionaryInfo != nil {
			in.SysDictionaryInfo.ID = 0
			if in.SysDictionaryInfo.Status == 0 {
				in.SysDictionaryInfo.Status = 1
			}
		}
		if err := validateItem(tx, in.SysDictionaryInfo); err != nil {
			return err
		}
		value := in.SysDictionaryInfo
		return tx.Create(&model.SysDictionaryInfo{Label: value.Label, Value: value.Value, Extend: value.Extend, Status: value.Status, Sort: value.Sort, SysDictionaryID: value.SysDictionaryID}).Error
	})
	return &pb.NoDataResponse{}, accessutil.FriendlyDuplicate(err)
}
