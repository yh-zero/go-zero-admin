package apilogic

import (
	"context"
	"github.com/zeromicro/go-zero/core/logx"
	"go-zero-admin/application/applet/rpc/internal/svc"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"
)

type DeleteApiLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	logx.Logger
}

func NewDeleteApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteApiLogic {
	return &DeleteApiLogic{ctx: ctx, svcCtx: svcCtx, Logger: logx.WithContext(ctx)}
}

func (l *DeleteApiLogic) DeleteApi(in *pb.DeleteApiRequest) (*pb.NoDataResponse, error) {
	if in.SysApi == nil {
		return nil, xerr.NewErrCodeMsg(xerr.REUQEST_PARAM_ERROR, "API不能为空")
	}
	return NewDeleteApisByIdsLogic(l.ctx, l.svcCtx).DeleteApisByIds(&pb.DeleteApisByIdsRequest{Ids: []int64{in.SysApi.ID}})
}
