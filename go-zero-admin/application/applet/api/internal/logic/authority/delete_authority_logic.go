package authority

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteAuthorityLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteAuthorityLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteAuthorityLogic {
	return &DeleteAuthorityLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteAuthorityLogic) DeleteAuthority(req *types.DeleteAuthorityRequest) (resp *types.MessageResponse, err error) {
	_, err = l.svcCtx.AppletAuthorityRPC.DeleteAuthority(l.ctx, &pb.DeleteAuthorityRequest{ID: req.ID})
	if err != nil {
		logx.Errorf("l.svcCtx.AppletAuthorityRPC.DeleteAuthority 错误 error: %v", err)
		return nil, err
	}

	return &types.MessageResponse{Message: "删除成功！"}, nil
}
