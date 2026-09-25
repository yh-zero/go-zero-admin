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

type UpdateSysDictionaryInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryInfoLogic {
	return &UpdateSysDictionaryInfoLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateSysDictionaryInfoLogic) UpdateSysDictionaryInfo(in *pb.UpdateSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	if in.SysDictionaryInfo == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "字典项不能为空")
	}
	err := l.svcCtx.DB.Transaction(func(tx *gorm.DB) error {
		var old model.SysDictionaryInfo
		if err := accessutil.RequireID(tx, &old, in.SysDictionaryInfo.ID); err != nil {
			return err
		}
		if err := validateItem(tx, in.SysDictionaryInfo); err != nil {
			return err
		}
		value := in.SysDictionaryInfo
		return tx.Model(&old).Updates(map[string]interface{}{"label": value.Label, "value": value.Value, "extend": value.Extend, "status": value.Status, "sort": value.Sort, "sys_dictionary_id": value.SysDictionaryID}).Error
	})
	return &pb.NoDataResponse{}, accessutil.FriendlyDuplicate(err)
}
