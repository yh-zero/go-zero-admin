package fileresourceservicelogic

import (
	"context"
	"errors"
	"path"
	"strings"

	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/datascope"
	"gorm.io/gorm"
)

type RegisterFileLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewRegisterFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterFileLogic {
	return &RegisterFileLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

func (l *RegisterFileLogic) RegisterFile(in *pb.RegisterFileRequest) (*pb.FileResource, error) {
	if in == nil {
		return nil, fileError("文件登记参数不能为空")
	}
	if err := validateFile(in.File); err != nil {
		return nil, err
	}
	if _, err := datascope.Apply(l.ctx, l.svcCtx.DB.Model(&model.SysFileResource{}), in.Actor, "owner_id", "department_id"); err != nil {
		return nil, err
	}
	file := model.SysFileResource{ObjectKey: in.File.ObjectKey, Name: path.Base(strings.ReplaceAll(in.File.Name, "\\", "/")), Mime: in.File.Mime,
		Size: in.File.Size, OwnerID: in.Actor.UserID, Visibility: in.File.Visibility, Status: "active"}
	err := l.svcCtx.DB.WithContext(l.ctx).Transaction(func(tx *gorm.DB) error {
		if err := accessutil.LockAdminGuard(tx); err != nil {
			return err
		}
		if _, err := datascope.Apply(l.ctx, tx.Model(&model.SysFileResource{}), in.Actor, "owner_id", "department_id"); err != nil {
			return err
		}
		var existing model.SysFileResource
		if err := tx.Where("object_key = ?", file.ObjectKey).First(&existing).Error; err == nil {
			if existing.OwnerID != file.OwnerID || existing.Name != file.Name || existing.Mime != file.Mime || existing.Size != file.Size || existing.Visibility != file.Visibility || existing.Status != "active" {
				return fileError("文件对象已登记且元数据不一致")
			}
			file = existing
			return nil
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		var membership model.SysUserDepartment
		if err := tx.Where("user_id = ?", in.Actor.UserID).First(&membership).Error; err == nil {
			file.DepartmentID = membership.DepartmentID
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err := tx.Create(&file).Error; err != nil {
			return err
		}
		return auditFile(l.ctx, tx, in.Actor, "registerFile", file.ID)
	})
	if err != nil {
		return nil, err
	}
	return fileProto(l.ctx, l.svcCtx.DB.DB, &file)
}
