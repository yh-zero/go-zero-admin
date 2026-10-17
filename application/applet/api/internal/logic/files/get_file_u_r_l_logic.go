// Code scaffolded by goctl. Safe to edit.

package files

import (
	"context"

	"go-zero-admin/application/applet/api/internal/svc"
	"go-zero-admin/application/applet/api/internal/types"
	"go-zero-admin/application/applet/rpc/pb"

	"github.com/zeromicro/go-zero/core/logx"
)

type GetFileURLLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// 获取受控文件访问地址。GetFileURL is a thin pass-through: the RPC tier decides
// whether to sign a private URL or return a public one.
func NewGetFileURLLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetFileURLLogic {
	return &GetFileURLLogic{Logger: logx.WithContext(ctx), ctx: ctx, svcCtx: svcCtx}
}

func (l *GetFileURLLogic) GetFileURL(req *types.FileURLRequest) (resp *types.FileURLResponse, err error) {
	resp2, err := l.svcCtx.AppletFileRPC.GetFileURL(l.ctx, &pb.FileIDRequest{Actor: actorSession(l.ctx), ID: req.Id})
	if err != nil {
		return nil, err
	}
	return &types.FileURLResponse{Url: resp2.Url, ExpiresIn: resp2.ExpiresIn}, nil
}
