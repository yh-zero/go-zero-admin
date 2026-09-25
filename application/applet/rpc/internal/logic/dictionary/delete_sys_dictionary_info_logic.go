package dictionarylogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type DeleteSysDictionaryInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteSysDictionaryInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteSysDictionaryInfoLogic {
	return &DeleteSysDictionaryInfoLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteSysDictionaryInfoLogic) DeleteSysDictionaryInfo(in *pb.DeleteSysDictionaryInfoRequest) (*pb.NoDataResponse, error) {
	var value model.SysDictionaryInfo
	if err := accessutil.RequireID(l.svcCtx.DB.DB, &value, in.ID); err != nil {
		return nil, err
	}
	return &pb.NoDataResponse{}, l.svcCtx.DB.Delete(&value).Error
}
