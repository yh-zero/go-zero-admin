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

type DeleteSysDictionaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSysDictionaryLogic {
	return &DeleteSysDictionaryLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteSysDictionaryLogic) DeleteSysDictionary(in *pb.DeleteSysDictionaryRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var value model.SysDictionary
		if err := accessutil.RequireID(tx, &value, in.ID); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.SysDictionaryInfo{}).Where("sys_dictionary_id = ?", in.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "字典仍有字典项，请先删除字典项")
		}
		return tx.Delete(&value).Error
	})
	return &pb.NoDataResponse{}, err
}
