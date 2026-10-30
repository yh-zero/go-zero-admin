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
	codes, err := l.svcCtx.SysMenuModel.FindPermCodesByUser(l.ctx, in.GetActor().GetUserId())
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.GetMyPermCodesResponse{Codes: codes}, nil
}
