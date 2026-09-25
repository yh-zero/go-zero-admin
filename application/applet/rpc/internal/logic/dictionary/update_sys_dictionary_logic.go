package dictionarylogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type UpdateSysDictionaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryLogic {
	return &UpdateSysDictionaryLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateSysDictionaryLogic) UpdateSysDictionary(in *pb.UpdateSysDictionaryRequest) (*pb.NoDataResponse, error) {
	if in.SysDictionary == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "字典不能为空")
	}
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var old model.SysDictionary
		if err := accessutil.RequireID(tx, &old, in.SysDictionary.ID); err != nil {
			return err
		}
		if err := validateDictionary(tx, in.SysDictionary); err != nil {
			return err
		}
		value := in.SysDictionary
		return tx.Model(&old).Updates(map[string]interface{}{"name": value.Name, "type": value.Type, "status": value.Status, "desc": value.Desc}).Error
	})
	return &pb.NoDataResponse{}, err
}
