package dictionarylogic

import (
	"context"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteSysDictionaryInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSysDictionaryInfoLogic {
	return &DeleteSysDictionaryInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 删除SysDictionaryInfo
func (l *DeleteSysDictionaryInfoLogic) DeleteSysDictionaryInfo(in *pb.DeleteSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	err := l.svcCtx.DB.Where("id = ?", in.ID).Delete(&model.SysDictionaryInfo{}).Error

	return &pb.NoDataResponse{}, err
}
