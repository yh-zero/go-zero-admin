package dictionarylogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateSysDictionaryLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateSysDictionaryLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateSysDictionaryLogic {
	return &UpdateSysDictionaryLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新SysDictionary
func (l *UpdateSysDictionaryLogic) UpdateSysDictionary(in *pb.UpdateSysDictionaryRequest) (*pb.NoDataResponse, error) {
	// todo: add your logic here and delete this line

	return &pb.NoDataResponse{}, nil
}
