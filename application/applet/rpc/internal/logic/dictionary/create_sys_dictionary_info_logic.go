package dictionarylogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateSysDictionaryInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateSysDictionaryInfoLogic {
	return &CreateSysDictionaryInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建SysDictionaryInfo
func (l *CreateSysDictionaryInfoLogic) CreateSysDictionaryInfo(in *pb.CreateSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	if in.SysDictionaryInfo.SysDictionaryID == 0 {
		return &pb.NoDataResponse{}, errors.New("错误： 父级id不能为空")
	}

	var modelSysDictionaryInfo model.SysDictionaryInfo
	_ = copier.Copy(&modelSysDictionaryInfo, in.SysDictionaryInfo)
	modelSysDictionaryInfo.DeletedAt.Valid = false
	err := l.svcCtx.DB.Create(&modelSysDictionaryInfo).Error

	return &pb.NoDataResponse{}, err
}
