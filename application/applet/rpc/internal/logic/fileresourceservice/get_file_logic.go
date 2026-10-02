package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileLogic {
	return &GetFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *GetFileLogic) GetFile(in *pb.FileIDRequest) (*pb.FileResource, error) {
	if in == nil {
		return nil, fileError("文件查询参数不能为空")
	}
	file, err := scopedFile(l.ctx, l.svcCtx.DB.DB, in.Actor, in.ID, false)
	if err != nil {
		return nil, err
	}
	if file.Status != "active" {
		return nil, fileError("文件已删除或正在删除")
	}
	return fileProto(l.ctx, l.svcCtx.DB.DB, file)
}
