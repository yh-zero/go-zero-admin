package apilogic

import (
	"context"
	"gorm.io/gorm"

	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/pkg/errors"
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
func (l *DeleteApiLogic) DeleteApi(in *pb.DeleteApiRequest) (*pb.NoDataResponse, error) {
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

	// 同步删除casbin对应策略 避免死策略残留 重建同路径api时旧角色自动获得权限
	if _, err = l.svcCtx.Casbin.RemoveFilteredPolicy(1, sysApi.Path, sysApi.Method); err != nil {
		logx.WithContext(l.ctx).Errorf("DeleteApi RemoveFilteredPolicy err: %v path: %s method: %s", err, sysApi.Path, sysApi.Method)
		return nil, err
	}

	return &pb.NoDataResponse{}, nil
}
