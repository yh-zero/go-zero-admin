package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type BeginDeleteFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewBeginDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BeginDeleteFileLogic {
	return &BeginDeleteFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *BeginDeleteFileLogic) BeginDeleteFile(in *pb.FileIDRequest) (*pb.FileResource, error) {
	if in == nil {
		return nil, fileError("文件删除参数不能为空")
	}
	var response *pb.FileResource
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		file, err := scopedFile(l.ctx, tx, in.Actor, in.ID, true)
		if err != nil {
			return err
		}
		referenced, err := hasLockedFileReferences(l.ctx, tx, file.ID)
		if err != nil {
			return err
		}
		if referenced {
			return fileError("文件仍被业务引用，不能删除")
		}
		response = fileProtoWithReferences(file, 0)
		if file.Status == "deleted" || file.Status == "deleting" {
			return nil
		}
		if file.Status != "active" {
			return fileError("文件状态无效")
		}
		if err := tx.Model(file).Update("status", "deleting").Error; err != nil {
			return err
		}
		response.Status = "deleting"
		return auditFile(l.ctx, tx, in.Actor, "beginDeleteFile", in.ID)
	})
	return response, err
}
