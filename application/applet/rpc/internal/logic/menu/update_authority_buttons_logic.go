package menulogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
	"gorm.io/gorm"
)

type UpdateAuthorityButtonsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAuthorityButtonsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAuthorityButtonsLogic {
	return &UpdateAuthorityButtonsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *UpdateAuthorityButtonsLogic) UpdateAuthorityButtons(in *pb.UpdateAuthorityButtonsRequest) (*pb.NoDataResponse, error) {
	ids, err := accessutil.UniqueIDs(in.MenuBtnIds)
	if err != nil {
		return nil, err
	}
	err = accessutil.AdminMenuTransaction(l.svcCtx.DB.WithContext(l.ctx), func(tx *gorm.DB) error {
		if err := accessutil.RequireRole(tx, in.AuthorityId); err != nil {
			return err
		}
		var definitions []model.SysBaseMenuBtn
		if len(ids) > 0 {
			if err := tx.Where("id IN ?", ids).Find(&definitions).Error; err != nil {
				return err
			}
		}
		if len(definitions) != len(ids) {
			return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "部分按钮不存在，授权未修改")
		}
		var menuIDs []int64
		if err := tx.Model(&model.SysAuthorityMenu{}).Where("sys_authority_authority_id = ?", in.AuthorityId).Pluck("sys_base_menu_id", &menuIDs).Error; err != nil {
			return err
		}
		allowed := map[int64]bool{}
		for _, id := range menuIDs {
			allowed[id] = true
		}
		rows := make([]model.SysAuthorityBtn, 0, len(definitions))
		for _, button := range definitions {
			if !allowed[button.SysBaseMenuID] {
				return xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "请先给角色分配按钮所属菜单")
			}
			rows = append(rows, model.SysAuthorityBtn{AuthorityId: in.AuthorityId, SysMenuID: button.SysBaseMenuID, SysBaseMenuBtnID: button.ID})
		}
		if err := tx.Where("authority_id = ?", in.AuthorityId).Delete(&model.SysAuthorityBtn{}).Error; err != nil {
			return err
		}
		if len(rows) > 0 {
			return tx.Create(&rows).Error
		}
		return nil
	})
	return &pb.NoDataResponse{}, err
}
