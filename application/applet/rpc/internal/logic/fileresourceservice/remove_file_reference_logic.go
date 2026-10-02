package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"
)

type RemoveFileReferenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRemoveFileReferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RemoveFileReferenceLogic {
	return &RemoveFileReferenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RemoveFileReferenceLogic) RemoveFileReference(in *pb.FileReferenceRequest) (*pb.NoDataResponse, error) {
	if err := validateReference(in); err != nil {
		return nil, err
	}
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if _, err := scopedFile(l.ctx, tx, in.Actor, in.FileID, true); err != nil {
			return err
		}
		result := tx.Where("file_id = ? AND object_type = ? AND object_id = ?", in.FileID, in.ObjectType, in.ObjectID).Delete(&model.SysFileReference{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return auditFile(l.ctx, tx, in.Actor, "removeFileReference", in.FileID)
	})
	return &pb.NoDataResponse{}, err
}
