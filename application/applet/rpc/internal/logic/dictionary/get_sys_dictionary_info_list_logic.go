package dictionarylogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
)

type GetSysDictionaryInfoListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetSysDictionaryInfoListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetSysDictionaryInfoListLogic {
	return &GetSysDictionaryInfoListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetSysDictionaryInfoListLogic) GetSysDictionaryInfoList(in *pb.GetSysDictionaryInfoListRequest) (*pb.GetSysDictionaryInfoListResponse, error) {
	if in.PageRequest == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "分页参数不能为空")
	}
	offset, size, err := accessutil.Page(in.PageRequest.PageNo, in.PageRequest.PageSize)
	if err != nil {
		return nil, err
	}
	db := l.svcCtx.DB.Model(&model.SysDictionaryInfo{})
	if in.SysDictionaryInfo == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "请选择所属字典")
	}
	var dictionary model.SysDictionary
	if err := accessutil.RequireID(l.svcCtx.DB.DB, &dictionary, in.SysDictionaryInfo.SysDictionaryID); err != nil {
		return nil, err
	}
	if value := in.SysDictionaryInfo; value != nil {
		if value.Label != "" {
			db = db.Where("label LIKE ?", "%"+value.Label+"%")
		}
		if in.HasValue {
			db = db.Where("value = ?", value.Value)
		}
		if value.Status != 0 {
			if err := accessutil.Status(value.Status); err != nil {
				return nil, err
			}
			db = db.Where("status = ?", value.Status)
		}
		if value.SysDictionaryID != 0 {
			db = db.Where("sys_dictionary_id = ?", value.SysDictionaryID)
		}
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var records []model.SysDictionaryInfo
	if err := db.Order("sort,id").Limit(size).Offset(offset).Find(&records).Error; err != nil {
		return nil, err
	}
	result := &pb.GetSysDictionaryInfoListResponse{Total: total}
	if err := copier.Copy(&result.SysDictionaryInfoList, records); err != nil {
		return nil, err
	}
	return result, nil
}
