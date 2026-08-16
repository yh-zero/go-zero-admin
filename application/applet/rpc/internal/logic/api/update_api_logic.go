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

type UpdateApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateApiLogic {
	return &UpdateApiLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新api
func (l *UpdateApiLogic) UpdateApi(in *pb.UpdateApiRequest) (*pb.NoDataResponse, error) {
	var modelSysApi model.SysApi
	var sysApi model.SysApi
	_ = copier.Copy(&sysApi, in.SysApi)
	err := l.svcCtx.DB.Where("id = ?", sysApi.ID).First(&modelSysApi).Error
	if modelSysApi.Path != sysApi.Path || modelSysApi.Method != sysApi.Method {
		if !errors.Is(l.svcCtx.DB.Where("path = ? AND method = ?", sysApi.Path, sysApi.Method).First(&model.SysApi{}).Error, gorm.ErrRecordNotFound) {
			return nil, errors.New("存在相同api路径或者请求method一样")
		}
	}
	if err != nil {
		return nil, err
	} else {
		err = l.UpdateCasbinApi(modelSysApi.Path, sysApi.Path, modelSysApi.Method, sysApi.Method)
		if err != nil {
			return nil, err
		}
		sysApi.DeletedAt.Valid = false
		err = l.svcCtx.DB.Save(&sysApi).Error
	}

	return &pb.NoDataResponse{}, nil
}

// UpdateCasbinApi 更新api后同步更新casbin策略 走enforcer写操作 自动落库并通过watcher广播
func (l *UpdateApiLogic) UpdateCasbinApi(oldPath string, newPath string, oldMethod string, newMethod string) error {
	oldRules := l.svcCtx.Casbin.GetFilteredPolicy(1, oldPath, oldMethod)
	if len(oldRules) == 0 {
		return nil
	}

	newRules := make([][]string, len(oldRules))
	for i, rule := range oldRules {
		newRules[i] = []string{rule[0], newPath, newMethod}
	}

	_, err := l.svcCtx.Casbin.UpdatePolicies(oldRules, newRules)
	return err
}
