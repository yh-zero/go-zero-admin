package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type FinishDeleteFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewFinishDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *FinishDeleteFileLogic {
	return &FinishDeleteFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *FinishDeleteFileLogic) FinishDeleteFile(in *pb.FileIDRequest) (*pb.NoDataResponse, error) {
	if in == nil {
		return nil, fileError("文件删除参数不能为空")
	}
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		file, err := scopedFile(l.ctx, tx, in.Actor, in.ID, true)
		if err != nil {
			return err
		}
		if file.Status == "deleted" {
			return nil
		}
		if file.Status != "deleting" {
			return fileError("文件没有进入待删除状态")
		}
		referenced, err := hasLockedFileReferences(l.ctx, tx, file.ID)
		if err != nil {
			return err
		}
		if referenced {
			return fileError("文件仍被引用，不能完成删除")
		}
		if err := tx.Model(file).Update("status", "deleted").Error; err != nil {
			return err
		}
		return auditFile(l.ctx, tx, in.Actor, "finishDeleteFile", in.ID)
	})
	return &pb.NoDataResponse{}, err
}
