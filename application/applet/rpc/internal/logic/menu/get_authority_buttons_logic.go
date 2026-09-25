package menulogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/logic/accessutil"
	"go-zero-admin/application/applet/rpc/internal/model"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
)

type GetAuthorityButtonsLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetAuthorityButtonsLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetAuthorityButtonsLogic {
	return &GetAuthorityButtonsLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetAuthorityButtonsLogic) GetAuthorityButtons(in *pb.GetAuthorityButtonsRequest) (*pb.GetAuthorityButtonsResponse, error) {
	if err := accessutil.RequireRole(l.svcCtx.DB.DB, in.AuthorityId); err != nil {
		return nil, err
	}
	ids := make([]int64, 0)
	if err := l.svcCtx.DB.Model(&model.SysAuthorityBtn{}).Where("authority_id = ?", in.AuthorityId).Order("sys_base_menu_btn_id").Pluck("sys_base_menu_btn_id", &ids).Error; err != nil {
		return nil, err
	}
	return &pb.GetAuthorityButtonsResponse{MenuBtnIds: ids}, nil
}
