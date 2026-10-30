package file

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	filepb "go-zero-admin/application/file/rpc/filepb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileURLLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetFileURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileURLLogic {
	return &GetFileURLLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetFileURLLogic) GetFileURL(req *types.GetFileURLReq) (resp *types.GetFileURLResp, err error) {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	rpcResp, err := l.svcCtx.FileRpc.GetFileURL(l.ctx, &filepb.GetFileURLRequest{
		Actor: &filepb.Actor{UserID: userID}, FileID: req.Id,
	})
	if err != nil {
		return nil, err
	}
	return &types.GetFileURLResp{URL: rpcResp.URL, ExpiresAt: rpcResp.ExpiresAt}, nil
}
