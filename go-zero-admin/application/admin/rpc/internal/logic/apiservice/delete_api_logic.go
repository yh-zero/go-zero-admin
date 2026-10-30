package apiservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApiLogic {
	return &DeleteApiLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteApiLogic) DeleteApi(in *adminpb.DeleteApiRequest) (*adminpb.NoDataResponse, error) {
	_, err := l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM sys_api WHERE id=?", in.ApiId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	l.svcCtx.PublishPolicyChanged(l.ctx)
	return &adminpb.NoDataResponse{}, nil
}
