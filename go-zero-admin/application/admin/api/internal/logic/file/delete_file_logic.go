package file

import (
	"context"

	"go-zero-admin/application/admin/api/internal/middleware"
	"go-zero-admin/application/admin/api/internal/svc"
	"go-zero-admin/application/admin/api/internal/types"
	filepb "go-zero-admin/application/file/rpc/filepb"

	"github.com/zeromicro/go-zero/core/logx"
)

type DeleteFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteFileLogic) DeleteFile(req *types.DeleteFileReq) error {
	userID := l.ctx.Value(middleware.CtxKeyUserID).(int64)
	_, err := l.svcCtx.FileRpc.DeleteFile(l.ctx, &filepb.DeleteFileRequest{
		Actor: &filepb.Actor{UserID: userID}, FileID: req.Id,
	})
	return err
}
