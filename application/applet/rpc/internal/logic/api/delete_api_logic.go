package apilogic

import (
	"context"
	"github.com/pkg/errors"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApiLogic {
	return &DeleteApiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除API列表
func (l *DeleteApiLogic) DeleteApi(in *pb.DeleteApiRequest) (*pb.DeleteApiResponse, error) {
	var sysApi model.SysApi
	if errors.Is(l.svcCtx.DB.Where("id = ?", in.SysApi.ID).First(&sysApi).Error, gorm.ErrRecordNotFound) {
		return nil, errors.New("api记录不存在")
	}

	//err := l.svcCtx.DB.Delete(&sysApi).Error // 软删除
	err := l.svcCtx.DB.Unscoped().Delete(&sysApi).Error //永久删除 硬删除
	if err != nil {
		logx.Errorf("l.svcCtx.DB.Delete(&sysApi) err: %v", err)
		return nil, err
	}

	// 这里要做删除功能  casbin 删除对应的策略 RemoveNamedPolicy()

	return &pb.DeleteApiResponse{}, nil
}
