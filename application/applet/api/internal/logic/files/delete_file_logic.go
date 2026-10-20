// Code scaffolded by goctl. Safe to edit.

package files

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"
	"go-zero-admin/pkg/filerpc"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/zrpc"
)

type DeleteFileLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 删除无引用文件，失败可重试。
// DeleteFile is a thin pass-through: the RPC tier handles begin -> object storage
// delete -> finish, and short-circuits on already-deleted rows.
func NewDeleteFileLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteFileLogic {
	return &DeleteFileLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *DeleteFileLogic) DeleteFile(req *types.FileIDRequest) (resp *types.MessageResponse, err error) {
	if _, err := l.svcCtx.AppletFileRPC.DeleteFile(l.ctx, &pb.FileIDRequest{Actor: actorSession(l.ctx), ID: req.Id}, zrpc.WithCallTimeout(filerpc.OperationTimeout)); err != nil {
		return nil, err
	}
	return &types.MessageResponse{Message: "文件已删除"}, nil
}
