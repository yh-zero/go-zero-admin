package dictservicelogic

import (
	"context"

	"github.com/zeromicro/go-zero/core/logx"

	adminpb "go-zero-admin/application/admin/rpc/adminpb"
	"go-zero-admin/application/admin/rpc/internal/svc"
	"go-zero-admin/pkg/xerr"
)

type DeleteDictLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteDictLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteDictLogic {
	return &DeleteDictLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteDictLogic) DeleteDict(in *adminpb.DeleteDictRequest) (*adminpb.NoDataResponse, error) {
	_, err := l.svcCtx.DB.ExecCtx(l.ctx, "UPDATE sys_dict SET deleted_at=NOW() WHERE id=?", in.DictId)
	if err != nil {
		return nil, xerr.NewErrCode(xerr.DB_ERROR)
	}
	return &adminpb.NoDataResponse{}, nil
}
