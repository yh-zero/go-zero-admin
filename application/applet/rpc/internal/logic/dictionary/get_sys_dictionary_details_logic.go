package dictionarylogic

import (
	"context"
	"errors"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type GetSysDictionaryDetailsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSysDictionaryDetailsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryDetailsLogic {
	return &GetSysDictionaryDetailsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetSysDictionaryDetailsLogic) GetSysDictionaryDetails(in *pb.GetSysDictionaryDetailsRequest) (*pb.GetSysDictionaryDetailsResponse, error) {
	if in.ID <= 0 && in.Type == "" {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "请提供字典ID或type")
	}
	status := in.Status
	if status == 0 {
		status = 1
	}
	if err := accessutil.Status(status); err != nil {
		return nil, err
	}
	db := l.svcCtx.DB.Where("status = ?", status)
	if in.ID > 0 {
		db = db.Where("id = ?", in.ID)
	} else {
		db = db.Where("type = ?", in.Type)
	}
	var record model.SysDictionary
	if err := db.Preload("SysDictionaryInfoList", func(tx *gorm.DB) *gorm.DB { return tx.Where("status = ?", 1).Order("sort,id") }).First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "字典不存在或状态不匹配")
		}
		return nil, err
	}
	result := &pb.SysDictionary{}
	if err := copier.Copy(result, record); err != nil {
		return nil, err
	}
	return &pb.GetSysDictionaryDetailsResponse{SysDictionary: result}, nil
}
