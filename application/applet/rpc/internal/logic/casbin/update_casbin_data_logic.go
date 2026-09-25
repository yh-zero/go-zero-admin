package casbinlogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"gorm.io/gorm"
)

type UpdateCasbinDataLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateCasbinDataLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateCasbinDataLogic {
	return &UpdateCasbinDataLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateCasbinDataLogic) UpdateCasbinData(in *pb.UpdateCasbinDataRequest) (*pb.NoDataResponse, error) {
	err := accessutil.PolicyTransaction(l.svcCtx, func(tx *gorm.DB) error { return replaceRolePolicies(tx, in.AuthorityId, in.CasbinInfoList) })
	return &pb.NoDataResponse{}, err
}
