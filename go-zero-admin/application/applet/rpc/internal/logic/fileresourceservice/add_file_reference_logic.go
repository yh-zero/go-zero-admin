package fileresourceservicelogic

import (
	"context"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AddFileReferenceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewAddFileReferenceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AddFileReferenceLogic {
	return &AddFileReferenceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *AddFileReferenceLogic) AddFileReference(in *pb.FileReferenceRequest) (*pb.NoDataResponse, error) {
	if err := validateReference(in); err != nil {
		return nil, err
	}
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		file, err := scopedFile(l.ctx, tx, in.Actor, in.FileID, true)
		if err != nil {
			return err
		}
		if file.Status != "active" {
			return fileError("文件非可用状态，不能添加引用")
		}
		reference := model.SysFileReference{FileID: in.FileID, ObjectType: in.ObjectType, ObjectID: in.ObjectID}
		result := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&reference)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		return auditFile(l.ctx, tx, in.Actor, "addFileReference", in.FileID)
	})
	return &pb.NoDataResponse{}, err
}
