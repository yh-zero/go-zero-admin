package apilogic

import (
	"context"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/jinzhu/copier"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateApiLogic {
	return &CreateApiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 创建/添加 API列表
func (l *CreateApiLogic) CreateApi(in *pb.CreateApiRequest) (*pb.NoDataResponse, error) {
	if !errors.Is(l.svcCtx.DB.Where("path = ? AND method = ?", in.SysApi.Path, in.SysApi.Method).First(&model.SysApi{}).Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("存在相同api")
	}
	var sysApi model.SysApi
	_ = copier.Copy(&sysApi, in.SysApi)
	//err := l.svcCtx.DB.Omit("deleted_at").Create(&in.SysApi).Error
	err := l.svcCtx.DB.Omit("deleted_at").Create(&sysApi).Error

	return &pb.NoDataResponse{}, err
}
