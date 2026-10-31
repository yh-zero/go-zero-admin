package menuservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type GetMyPermCodesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewGetMyPermCodesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetMyPermCodesLogic {
	return &GetMyPermCodesLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *GetMyPermCodesLogic) GetMyPermCodes(in *adminpb.GetMyPermCodesRequest) (*adminpb.GetMyPermCodesResponse, error) {
	roleCodes, err := l.svcCtx.SysMenuModel.ListRoleCodesByUser(l.ctx, in.Actor.UserId)
	if err != nil {
		l.Errorf("query user roles error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	if len(roleCodes) == 0 {
		return &adminpb.GetMyPermCodesResponse{Codes: []string{}}, nil
	}
	codes, err := l.svcCtx.SysMenuModel.ListPermCodesByRoleCodes(l.ctx, roleCodes)
	if err != nil {
		l.Errorf("query perm codes error: %v", err)
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMyPermCodesResponse{Codes: codes}, nil
}
