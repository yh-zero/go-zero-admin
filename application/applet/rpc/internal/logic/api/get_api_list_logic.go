package apilogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type GetApiListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetApiListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApiListLogic {
	return &GetApiListLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 获取API列表
func (l *GetApiListLogic) GetApiList(in *pb.GetApiListRequest) (*pb.GetApiListResponse, error) {
	offset := in.PageRequest.PageSize * (in.PageRequest.PageNo - 1)
	db := l.svcCtx.DB.Model(&model.SysApi{})
	var apiList []model.SysApi
	var total int64

	// 处理参数
	if in.SysApi.Path != "" {
		db = db.Where("path LIKE ?", "%"+in.SysApi.Path+"%")
	}
	if in.SysApi.Description != "" {
		db = db.Where("description LIKE ?", "%"+in.SysApi.Description+"%")
	}
	if in.SysApi.Method != "" {
		db = db.Where("method = ?", in.SysApi.Method)
	}
	if in.SysApi.ApiGroup != "" { // 改包含
		db = db.Where("api_group LIKE ?", "%"+in.SysApi.ApiGroup+"%")
		//db = db.Where("api_group = ?", in.SysApi.ApiGroup)
	}
	err := db.Count(&total).Error
	if err != nil {
		return nil, err
	} else {
		db = db.Limit(int(in.PageRequest.PageSize)).Offset(int(offset))
		if in.OrderKey != "" {
			// 设置有效排序key 防止sql注入
			var OrderStr string
			orderMap := make(map[string]bool, 5)
			orderMap["id"] = true
			orderMap["path"] = true
			orderMap["api_group"] = true
			orderMap["description"] = true
			orderMap["method"] = true
			if orderMap[in.OrderKey] {
				if in.Desc {
					OrderStr = in.OrderKey + " desc"
				} else {
					OrderStr = in.OrderKey
				}
			} else {
				//与orderMap中的任何订单键都不匹配 则为非法字段
				return nil, err
			}
			err = db.Order(OrderStr).Find(&apiList).Error
		} else {
			//err = db.Order("api_group").Find(&apiList).Error
			err = db.Order("api_group").Find(&apiList).Error
		}
	}

	var pbSysApi []*pb.SysApi
	_ = copier.Copy(&pbSysApi, apiList)

	return &pb.GetApiListResponse{
		SysApi: pbSysApi,
		Total:  total,
	}, nil
}
