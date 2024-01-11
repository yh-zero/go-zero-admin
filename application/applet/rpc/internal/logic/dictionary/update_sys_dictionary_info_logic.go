package dictionarylogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/model"
	"reflect"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSysDictionaryInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryInfoLogic {
	return &UpdateSysDictionaryInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新SysDictionaryInfo
func (l *UpdateSysDictionaryInfoLogic) UpdateSysDictionaryInfo(in *pb.UpdateSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	// 构造要更新的字段和值的 map
	sysDictionaryInfo := map[string]interface{}{}

	// 判断字段是否为空，不为空则添加到 map 中
	addIfNotEmpty("Label", in.SysDictionaryInfo.Label, sysDictionaryInfo)
	addIfNotEmpty("Value", in.SysDictionaryInfo.Value, sysDictionaryInfo)
	addIfNotEmpty("Extend", in.SysDictionaryInfo.Extend, sysDictionaryInfo)
	addIfNotEmpty("Status", in.SysDictionaryInfo.Status, sysDictionaryInfo)
	addIfNotEmpty("Sort", in.SysDictionaryInfo.Sort, sysDictionaryInfo)

	// 检查是否有任何需要更新的字段
	if len(sysDictionaryInfo) == 0 {
		// 没有需要更新的字段，直接返回
		logx.Error("No fields to update.")
		return &pb.NoDataResponse{}, nil
	}

	// 使用 Update 方法更新指定 id 的记录
	err := l.svcCtx.DB.Model(model.SysDictionaryInfo{}).
		Where("id = ?", in.SysDictionaryInfo.ID).
		Updates(sysDictionaryInfo).Error

	// 检查是否有错误，并返回相应的错误消息
	if err != nil {
		return nil, err
	}

	return &pb.NoDataResponse{}, nil
}

func addIfNotEmpty(fieldName string, fieldValue interface{}, updateMap map[string]interface{}) {
	if fieldValue != nil && !reflect.ValueOf(fieldValue).IsZero() {
		updateMap[fieldName] = fieldValue
	}
}
