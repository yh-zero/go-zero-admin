package apilogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type CreateApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewCreateApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateApiLogic {
	return &CreateApiLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *CreateApiLogic) CreateApi(in *pb.CreateApiRequest) (*pb.NoDataResponse, error) {
	if in.SysApi == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "API不能为空")
	}
	path, method, err := accessutil.API(in.SysApi.Path, in.SysApi.Method)
	if err != nil {
		return nil, err
	}
	err = accessutil.PolicyTransaction(l.svcCtx, func(tx *gorm.DB) error {
		if err := accessutil.Unique(tx, &model.SysApi{}, "path = ? AND method = ?", path, method); err != nil {
			return err
		}
		return tx.Create(&model.SysApi{Path: path, Method: method, ApiGroup: in.SysApi.ApiGroup, Description: in.SysApi.Description}).Error
	})
	return &pb.NoDataResponse{}, err
}
