package authoritylogic

import (
	"context"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
	"strconv"
)

type DeleteAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAuthorityLogic {
	return &DeleteAuthorityLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteAuthorityLogic) DeleteAuthority(in *pb.DeleteAuthorityRequest) (*pb.NoDataResponse, error) {
	if in.ID == accessutil.AdminAuthorityID {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "内置管理员角色不能删除")
	}
	err := accessutil.PolicyTransaction(l.svcCtx, func(tx *gorm.DB) error {
		if err := accessutil.RequireRole(tx, in.ID); err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&model.SysAuthority{}).Where("parent_id = ? AND deleted_at IS NULL", in.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "此角色存在子角色，不允许删除")
		}
		if err := tx.Model(&model.SysUser{}).Where("authority_id = ?", in.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "此角色有用户正在使用，不允许删除")
		}
		if err := tx.Model(&model.SysUserAuthority{}).Where("sys_authority_authority_id = ?", in.ID).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "此角色有用户正在使用，不允许删除")
		}
		if err := tx.Where("sys_authority_authority_id = ?", in.ID).Delete(&model.SysAuthorityMenu{}).Error; err != nil {
			return err
		}
		if err := tx.Where("authority_id = ?", in.ID).Delete(&model.SysAuthorityBtn{}).Error; err != nil {
			return err
		}
		if err := tx.Table("sys_data_authority_id").Where("sys_authority_authority_id = ? OR data_authority_id_authority_id = ?", in.ID, in.ID).Delete(nil).Error; err != nil {
			return err
		}
		if err := tx.Where("ptype = ? AND v0 = ?", "p", strconv.FormatInt(in.ID, 10)).Delete(&gormadapter.CasbinRule{}).Error; err != nil {
			return err
		}
		return tx.Where("authority_id = ?", in.ID).Delete(&model.SysAuthority{}).Error
	})
	return &pb.NoDataResponse{}, err
}
