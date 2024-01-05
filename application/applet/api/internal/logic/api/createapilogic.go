package api

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/result/xerr"

	"github.com/jinzhu/copier"
	"github.com/zeromicro/go-zero/core/logx"
)

type CreateApiLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateApiLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateApiLogic {
	return &CreateApiLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateApiLogic) CreateApi(req *types.CreateApiRequest) (resp *types.CreateApiResponse, err error) {
	var sysApi pb.SysApi
	_ = copier.Copy(&sysApi, req.SysApi)
	_, err = l.svcCtx.AppletAPIRPC.CreateApi(l.ctx, &pb.CreateApiRequest{SysApi: &sysApi})
	if err != nil {
		logx.Errorf("创建失败 err: %v", err)
		return nil, xerr.NewErrMsg("创建失败不能拥有同样的属性")
	}

	return &types.CreateApiResponse{Message: "创建成功"}, nil
}
