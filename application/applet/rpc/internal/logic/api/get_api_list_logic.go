package apilogic

import (
	"context"
	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"strings"
)

type GetApiListLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetApiListLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetApiListLogic {
	return &GetApiListLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetApiListLogic) GetApiList(in *pb.GetApiListRequest) (*pb.GetApiListResponse, error) {
	if in.PageRequest == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "分页参数不能为空")
	}
	offset, size, err := accessutil.Page(in.PageRequest.PageNo, in.PageRequest.PageSize)
	if err != nil {
		return nil, err
	}
	order := in.OrderKey
	if order == "" {
		order = "id"
	}
	allowed := map[string]bool{"id": true, "path": true, "api_group": true, "description": true, "method": true}
	if !allowed[order] {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "不支持的排序字段")
	}
	if in.Desc {
		order += " desc"
	}
	db := l.svcCtx.DB.Model(&model.SysApi{})
	if api := in.SysApi; api != nil {
		if api.Path != "" {
			db = db.Where("path LIKE ?", "%"+api.Path+"%")
		}
		if api.Description != "" {
			db = db.Where("description LIKE ?", "%"+api.Description+"%")
		}
		if api.ApiGroup != "" {
			db = db.Where("api_group LIKE ?", "%"+api.ApiGroup+"%")
		}
		if api.Method != "" {
			db = db.Where("method = ?", strings.ToUpper(api.Method))
		}
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var records []model.SysApi
	if err := db.Order(order).Limit(size).Offset(offset).Find(&records).Error; err != nil {
		return nil, err
	}
	result := &pb.GetApiListResponse{Total: total}
	if err := copier.Copy(&result.SysApi, records); err != nil {
		return nil, err
	}
	return result, nil
}
