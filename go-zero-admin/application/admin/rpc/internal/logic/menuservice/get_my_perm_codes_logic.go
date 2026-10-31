package menuservicelogic

import (
	"context"
	"fmt"

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
	var roleCodes []string
	err := l.svcCtx.DB.QueryRowsCtx(l.ctx, &roleCodes,
		"SELECT v1 FROM casbin_rule WHERE ptype='g' AND v0=?", fmt.Sprintf("user:%d", in.Actor.UserId))
	if err != nil || len(roleCodes) == 0 {
		return &adminpb.GetMyPermCodesResponse{Codes: []string{}}, nil
	}
	var codes []string
	err = l.svcCtx.DB.QueryRowsCtx(l.ctx, &codes,
		"SELECT DISTINCT m.perm_code FROM sys_menu m JOIN sys_role_menu rm ON m.id=rm.menu_id JOIN sys_role r ON rm.role_id=r.id "+
			"WHERE r.code IN (?) AND m.type=3 AND m.perm_code != '' AND m.deleted_at IS NULL",
		roleCodes)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMyPermCodesResponse{Codes: codes}, nil
}
