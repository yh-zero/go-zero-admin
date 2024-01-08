package authoritylogic

import (
	"context"
	"fmt"
	"github.com/jinzhu/copier"
	"go-zero-admin/application/applet/rpc/internal/model"

	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type UpdateAuthorityLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewUpdateAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateAuthorityLogic {
	return &UpdateAuthorityLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
		Logger: logx.WithContext(ctx),
	}
}

// 更新角色 -- 设为首页
func (l *UpdateAuthorityLogic) UpdateAuthority(in *pb.UpdateAuthorityRequest) (*pb.UpdateAuthorityResponse, error) {
	var sysAuthority model.SysAuthority

	_ = copier.Copy(&sysAuthority, in.SysAuthority)
	err := l.svcCtx.DB.Where("authority_id = ?", sysAuthority.AuthorityId).First(&model.SysAuthority{}).Omit("deleted_at").Updates(&sysAuthority).Error
	fmt.Println("---------- sysAuthority", sysAuthority)

	var pbSysAuthority pb.SysAuthority
	_ = copier.Copy(&pbSysAuthority, sysAuthority)
	fmt.Println("---------- pbSysAuthority", pbSysAuthority)

	return &pb.UpdateAuthorityResponse{SysAuthority: &pbSysAuthority}, err
}
